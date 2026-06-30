package main

import (
	"context"
	"log"

	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters/testdata/workerdemo/consumer"
)

func main() {
	c := consumer.NewOrderConsumer("jobs")
	if err := c.Process(context.Background()); err != nil {
		log.Fatal(err)
	}
}
