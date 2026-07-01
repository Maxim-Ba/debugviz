package debugviz

import (
	"context"
	"crypto/rand"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

type contextKey int

const spanContextKey contextKey = 1

// Span is an in-flight trace span attached to context.
type Span struct {
	TraceID      string
	SpanID       string
	ParentSpanID *string
	Name         string
	File         string
	Line         int
	Status       protocol.SpanStatus
	EntryKind    protocol.EntryKind
	EntryName    string

	startTime time.Time
	isRoot    bool
	record    bool
}

// SpanFromContext returns the active span or nil when tracing is disabled.
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	span, _ := ctx.Value(spanContextKey).(*Span)
	return span
}

// StartSpan starts a child span and returns an updated context plus an end callback.
// When tracing is disabled, it returns the input context and a no-op end function.
func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if !isEnabled() {
		return ctx, func() {}
	}

	parent := SpanFromContext(ctx)
	if parent != nil && !parent.record {
		return ctx, func() {}
	}

	file, line := callerLocation(2)
	if line < 1 {
		line = 1
	}
	span := &Span{
		Name:      name,
		File:      file,
		Line:      line,
		Status:    protocol.SpanStatusOK,
		startTime: time.Now(),
		record:    true,
	}

	if parent != nil {
		span.TraceID = parent.TraceID
		span.SpanID = newSpanID()
		parentID := parent.SpanID
		span.ParentSpanID = &parentID
	} else {
		span.TraceID = newTraceID()
		span.SpanID = newSpanID()
	}

	return context.WithValue(ctx, spanContextKey, span), func() {
		endSpan(span)
	}
}

func startRootSpan(ctx context.Context, kind protocol.EntryKind, entryName string) (context.Context, func()) {
	if !isEnabled() {
		return ctx, func() {}
	}

	file, line := callerLocation(2)
	if line < 1 {
		line = 1
	}
	traceID := traceIDFromContext(ctx)
	if traceID == "" {
		traceID = newTraceID()
	}

	span := &Span{
		TraceID:   traceID,
		SpanID:    newSpanID(),
		Name:      entryName,
		File:      file,
		Line:      line,
		Status:    protocol.SpanStatusOK,
		EntryKind: kind,
		EntryName: entryName,
		startTime: time.Now(),
		isRoot:    true,
		record:    shouldRecordTrace(),
	}

	return context.WithValue(ctx, spanContextKey, span), func() {
		endSpan(span)
	}
}

func endSpan(span *Span) {
	if span == nil || !span.record {
		return
	}

	duration := time.Since(span.startTime)
	event := protocol.TraceEvent{
		TraceID:      span.TraceID,
		SpanID:       span.SpanID,
		ParentSpanID: span.ParentSpanID,
		Name:         span.Name,
		File:         span.File,
		Line:         span.Line,
		StartUs:      span.startTime.UnixMicro(),
		DurationUs:   duration.Microseconds(),
		Status:       span.Status,
	}
	if span.isRoot {
		event.EntryKind = span.EntryKind
		event.EntryName = span.EntryName
	}

	if exp := currentExporter(); exp != nil {
		exp.enqueue(event)
	}
}

func callerLocation(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}
	return relativizeSourcePath(filepath.ToSlash(file)), line
}

func relativizeSourcePath(path string) string {
	if sourceRoot != "" {
		root := filepath.ToSlash(sourceRoot)
		if strings.HasPrefix(path, root+"/") {
			return strings.TrimPrefix(path, root+"/")
		}
	}
	for _, marker := range []string{"demo/", "go/lib/", "internal/", "pkg/", "cmd/"} {
		if i := strings.Index(path, marker); i >= 0 {
			return path[i:]
		}
	}
	return path
}

func shouldRecordTrace() bool {
	rate := currentSampleRate()
	if rate >= 1.0 {
		return true
	}
	if rate <= 0 {
		return false
	}
	var b [1]byte
	_, _ = rand.Read(b[:])
	return float64(b[0])/255.0 < rate
}
