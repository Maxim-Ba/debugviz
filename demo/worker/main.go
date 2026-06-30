package main

import (
	"context"
	"log"
	"time"

	"github.com/Maxim-Ba/debugviz/demo/worker/internal/consumer"
)

func main() {
	c := consumer.NewOrderConsumer("orders")
	for {
		if err := c.Process(context.Background()); err != nil {
			log.Println(err)
		}
		time.Sleep(2 * time.Second)
	}
}
