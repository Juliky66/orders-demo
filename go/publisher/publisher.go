package main

import (
	"fmt"
	"os"

	"github.com/nats-io/stan.go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: publisher <path-to-json>")
		return
	}
	payload, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	stanConn, err := stan.Connect("orders-cluster", "publisher-cli", stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		panic(err)
	}
	defer stanConn.Close()

	if err := stanConn.Publish("orders", payload); err != nil {
		panic(err)
	}
	fmt.Println("published", os.Args[1])
}
