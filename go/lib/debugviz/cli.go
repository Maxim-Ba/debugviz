package debugviz

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

type cliEntryNameKey struct{}

// RunCLI executes a CLI application function with a root span.
// Use CLICommandPreRun on the root cobra command to set entry_name from the command path.
func RunCLI(appName string, fn func(ctx context.Context) error) error {
	if !isEnabled() {
		return fn(context.Background())
	}

	ctx := context.WithValue(context.Background(), cliEntryNameKey{}, new(string))
	ctx, end := startRootSpan(ctx, protocol.EntryKindCLI, appName)
	defer end()

	err := fn(ctx)
	if span := SpanFromContext(ctx); span != nil {
		if namePtr, ok := ctx.Value(cliEntryNameKey{}).(*string); ok && namePtr != nil && *namePtr != "" {
			span.EntryName = *namePtr
			span.Name = *namePtr
		}
	}
	if span := SpanFromContext(ctx); span != nil && err != nil {
		span.Status = protocol.SpanStatusError
	}
	return err
}

// CLICommandPreRun updates the active CLI root span entry_name from cobra command path.
func CLICommandPreRun(cmd *cobra.Command, _ []string) {
	span := SpanFromContext(cmd.Context())
	if span == nil {
		return
	}
	path := strings.TrimSpace(cmd.CommandPath())
	if path == "" {
		return
	}
	span.EntryName = path
	span.Name = path
	if namePtr, ok := cmd.Context().Value(cliEntryNameKey{}).(*string); ok && namePtr != nil {
		*namePtr = path
	}
}
