// Package main usage of singleflight.Group
package main

import (
	"fmt"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"singleflight"
)

// Arg is the argument of Data.GetData
type Arg struct {
	Caller int
}

// Data is the RPC server struct
type Data struct{}

// GetData is the RPC server
func (d *Data) GetData(arg *Arg, reply *string) error {
	fmt.Printf("Request from client %d\n", arg.Caller)
	time.Sleep(time.Second)
	*reply = "source Data from RPC server"
	return nil
}

// main is the entry point of the program.
func main() {
	// create the RPC server
	d := new(Data)
	rpc.Register(d)
	rpc.HandleHTTP()
	fmt.Println("Starting RPC server...")
	go call()
	// start the RPC server
	if err := http.ListenAndServe(":1234", nil); err != nil {
		panic(err)
	}
}

// call is the RPC client
func call() {
	// wait for the RPC server to start
	time.Sleep(1 * time.Second)
	// create the RPC client
	ent, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		panic(err)
	}
	sf := new(singleflight.Group)
	wg := sync.WaitGroup{}
	wg.Add(100)
	// 100 goroutines will call synced the RPC server
	for i := 0; i < 100; i++ {
		fn := func() (interface{}, error) {
			var reply string
			err := ent.Call("Data.GetData", &Arg{Caller: i}, &reply)
			if err != nil {
				return nil, err
			}
			return reply, nil
		}
		// call singleflight.Do in goroutine
		go func(i int) {
			result, _ := sf.Do("foo", fn)
			fmt.Printf("caller: %d get result: %s\n", i, result)
			wg.Done()
		}(i)
	}
}
