package consumer

import "context"

type OrderConsumer struct {
	queue string
}

func NewOrderConsumer(queue string) *OrderConsumer {
	return &OrderConsumer{queue: queue}
}

func (c *OrderConsumer) Process(ctx context.Context) error {
	return nil
}

func (c *OrderConsumer) Handle(ctx context.Context, payload string) error {
	return nil
}
