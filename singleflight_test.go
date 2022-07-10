package singleflight

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	var g Group
	v, err := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})
	if got, want := fmt.Sprintf("%v (%T)", v, v), "bar (string)"; got != want {
		t.Errorf("Do = %v; want %v", got, want)
	}
	if err != nil {
		t.Errorf("Do error = %v", err)
	}
}

func TestDoErr(t *testing.T) {
	var g Group
	someErr := errors.New("some error")
	v, err := g.Do("key", func() (interface{}, error) {
		return nil, someErr
	})
	if err != someErr {
		t.Errorf("Do error = %v; want someErr", err)
	}
	if v != nil {
		t.Errorf("unexpected non-nil value %#v", v)
	}
}

func TestDoDupSuppress(t *testing.T) {
	var g Group
	c := make(chan string)
	var calls int32
	fn := func() (interface{}, error) {
		atomic.AddInt32(&calls, 1)
		return <-c, nil
	}

	const n = 10
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			v, err := g.Do("key", fn)
			if err != nil {
				t.Errorf("Do error: %v", err)
			}
			if v.(string) != "bar" {
				t.Errorf("got %q; want %q", v, "bar")
			}
			wg.Done()
		}()
	}
	time.Sleep(100 * time.Millisecond) // let goroutines above block
	c <- "bar"
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("number of calls = %d; want 1", got)
	}
}

func TestGroupDo(t *testing.T) {
	type fields struct {
		mu sync.Mutex
		m  map[string]*call
	}
	type args struct {
		key string
		fn  func() (interface{}, error)
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "duplicate",
			fields: fields{
				m: map[string]*call{
					"key": {
						wg: sync.WaitGroup{},
					},
				},
			},
			args: args{
				key: "key",
				fn: func() (interface{}, error) {
					return "val", nil
				},
			},
			want:    "val",
			wantErr: false,
		},
	}
	const n = 100
	for i := 0; i < n; i++ {
		go func() {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					g := &Group{
						mu: tt.fields.mu,
						m:  tt.fields.m,
					}
					got, err := g.Do(tt.args.key, tt.args.fn)
					if (err != nil) != tt.wantErr {
						t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
						return
					}
					if !reflect.DeepEqual(got, tt.want) {
						t.Errorf("Do() got = %v, want %v", got, tt.want)
					}
				})
			}
		}()
	}
}
