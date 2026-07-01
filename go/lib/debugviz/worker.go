package debugviz

import (
	"context"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// RunJob executes a worker job function with a root span.
func RunJob(ctx context.Context, jobName string, fn func(ctx context.Context) error) error {
	if !isEnabled() {
		return fn(ctx)
	}

	ctx, end := startRootSpan(ctx, protocol.EntryKindWorker, jobName)
	defer end()

	err := fn(ctx)
	if span := SpanFromContext(ctx); span != nil && err != nil {
		span.Status = protocol.SpanStatusError
	}
	return err
}
