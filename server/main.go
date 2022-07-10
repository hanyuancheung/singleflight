package main

import (
	"fmt"
	"net/http"
	"net/rpc"
	"singleflight"
	"sync"
	"time"
)

type Arg struct {
	Caller int
}

type Data struct{}

func (d *Data) GetData(arg *Arg, reply *string) error {
	fmt.Printf("Request from client %d\n", arg.Caller)
	time.Sleep(time.Second)
	*reply = "source Data from RPC server"
	return nil
}

func main() {
	d := new(Data)
	rpc.Register(d)
	rpc.HandleHTTP()
	fmt.Println("Starting RPC server...")
	go call()
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

func call() {
	time.Sleep(1 * time.Second)

	ent, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		panic(err)
	}
	sf := new(singleflight.Group)
	wg := sync.WaitGroup{}
	wg.Add(100)

	for i := 0; i < 100; i++ {
		fn := func() (interface{}, error) {
			var reply string
			err := ent.Call("Data.GetData", &Arg{Caller: i}, &reply)
			if err != nil {
				return nil, err
			}
			return reply, nil
		}

		go func(i int) {
			result, _ := sf.Do("foo", fn)
			fmt.Printf("caller: %d get result: %s\n", i, result)
			wg.Done()
		}(i)
	}
}
