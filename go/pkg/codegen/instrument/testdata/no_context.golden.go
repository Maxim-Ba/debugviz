package worker

import (
	"context"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

func processOrder(orderID string) error {
	_, __dv_end := debugviz.StartSpan(context.Background(), "worker.processOrder")
	defer __dv_end()
	return nil
}
