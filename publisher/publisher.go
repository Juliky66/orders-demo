package main

import (
	"io/ioutil"
	"log"
	"os"

	stan "github.com/nats-io/stan.go"
)

func main() {
	fn := "model.json"
	if len(os.Args) > 1 {
		fn = os.Args[1]
	}
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal(err)
	}

	sc, err := stan.Connect("orders-cluster", "publisher-cli", stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	if err := sc.Publish("orders", data); err != nil {
		log.Fatal(err)
	}
	log.Println("published", fn)
}
