package main

import (
	"context"
	"log"
	"time"
)

type consumer struct{}

func (c *consumer) Process(ctx context.Context) error {
	return nil
}

func main() {
	c := &consumer{}
	for {
		err := c.Process(context.Background())
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Second)
	}
}
