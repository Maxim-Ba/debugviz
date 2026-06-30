package consumer

import "context"

type OrderConsumer struct {
	queue string
}

func NewOrderConsumer(queue string) *OrderConsumer {
	return &OrderConsumer{queue: queue}
}

func (c *OrderConsumer) Process(ctx context.Context) error {
	_ = c.queue
	return nil
}
