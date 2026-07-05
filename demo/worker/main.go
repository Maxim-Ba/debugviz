package main

import (
	"context"
	"log"
	"time"

	"github.com/Maxim-Ba/debugviz/demo/worker/internal/consumer"
)

//debugviz:app name=demo-worker
//go:generate go run ../../go/cmd/debugviz wire --config debugviz.yaml --write .

func main() {
	c := consumer.NewOrderConsumer("orders")
	for {
		err := c.Process(context.Background())
		if err != nil {
			log.Println(err)
		}
		time.Sleep(2 * time.Second)
	}
}
