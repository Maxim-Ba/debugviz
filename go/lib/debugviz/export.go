package debugviz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

const ringBufferCapacity = 1000

type exporter struct {
	endpoint   string
	batchSize  int
	client     *http.Client
	ring       *spanRing
	pending    []protocol.TraceEvent
	notify     chan struct{}
	stop       chan struct{}
	wg         sync.WaitGroup
	flushEvery time.Duration
}

func newExporter(cfg Config) (*exporter, error) {
	endpoint, err := spansEndpoint(cfg.ServerURL)
	if err != nil {
		return nil, err
	}

	exp := &exporter{
		endpoint:   endpoint,
		batchSize:  cfg.BatchSize,
		client:     &http.Client{Timeout: 5 * time.Second},
		ring:       newSpanRing(ringBufferCapacity),
		pending:    make([]protocol.TraceEvent, 0, cfg.BatchSize),
		notify:     make(chan struct{}, 1),
		stop:       make(chan struct{}),
		flushEvery: 200 * time.Millisecond,
	}
	exp.wg.Add(1)
	go exp.loop()
	return exp, nil
}

func (e *exporter) enqueue(event protocol.TraceEvent) {
	e.ring.push(event)
	select {
	case e.notify <- struct{}{}:
	default:
	}
}

func (e *exporter) loop() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.flushEvery)
	defer ticker.Stop()

	for {
		select {
		case <-e.stop:
			e.drainRing()
			e.flush()
			return
		case <-e.notify:
			e.drainRing()
			if len(e.pending) >= e.batchSize {
				e.flush()
			}
		case <-ticker.C:
			e.drainRing()
			if len(e.pending) > 0 {
				e.flush()
			}
		}
	}
}

func (e *exporter) drainRing() {
	for {
		event, ok := e.ring.pop()
		if !ok {
			return
		}
		e.pending = append(e.pending, event)
	}
}

func (e *exporter) flush() {
	if len(e.pending) == 0 {
		return
	}
	batch := e.pending
	e.pending = make([]protocol.TraceEvent, 0, e.batchSize)

	if err := e.postBatch(batch); err != nil {
		for _, event := range batch {
			e.ring.push(event)
		}
	}
}

func (e *exporter) postBatch(batch []protocol.TraceEvent) error {
	body, err := json.Marshal(map[string]any{"spans": batch})
	if err != nil {
		return err
	}

	var lastErr error
	backoff := 100 * time.Millisecond
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, e.endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("debugviz exporter: status %d", resp.StatusCode)
	}
	return lastErr
}

func (e *exporter) shutdown() {
	close(e.stop)
	e.wg.Wait()
}

type spanRing struct {
	mu    sync.Mutex
	buf   []protocol.TraceEvent
	head  int
	tail  int
	count int
	cap   int
}

func newSpanRing(capacity int) *spanRing {
	return &spanRing{
		buf: make([]protocol.TraceEvent, capacity),
		cap: capacity,
	}
}

func (r *spanRing) push(event protocol.TraceEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count == r.cap {
		r.head = (r.head + 1) % r.cap
		r.count--
	}
	r.buf[r.tail] = event
	r.tail = (r.tail + 1) % r.cap
	r.count++
}

func (r *spanRing) pop() (protocol.TraceEvent, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count == 0 {
		return protocol.TraceEvent{}, false
	}
	event := r.buf[r.head]
	r.head = (r.head + 1) % r.cap
	r.count--
	return event, true
}
