package main

import (
	"context"
	"log"
	"time"

	"github.com/Maxim-Ba/debugviz/demo/worker/internal/consumer"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}

	c := consumer.NewOrderConsumer("orders")
	for {
		err := debugviz.RunJob(context.Background(), "OrderConsumer.Process", func(ctx context.Context) error {
			return c.Process(ctx)
		})
		if err != nil {
			log.Println(err)
		}
		time.Sleep(2 * time.Second)
	}
}
