package main

import (
	"context"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"log"
	"time"
)

type consumer struct{}

func (c *consumer) Process(ctx context.Context) error {
	return nil
}

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	c := &consumer{}
	for {
		err := debugviz.RunJob(context.Background(), "OrderConsumer.Process", func(ctx context.Context) error {
			return c.Process(ctx)
		})
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Second)
	}
}
