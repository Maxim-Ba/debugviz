package consumer

import (
	"context"
	"time"
)

type OrderConsumer struct {
	queue string
}

func NewOrderConsumer(queue string) *OrderConsumer {
	return &OrderConsumer{queue: queue}
}

func (c *OrderConsumer) Process(ctx context.Context) error {
	_ = ctx
	_ = c.queue
	time.Sleep(50 * time.Millisecond)
	return nil
}
