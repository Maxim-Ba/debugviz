import type { EntryKind, SpanStatus, TraceEvent } from "@debugviz/protocol";

/** Span stored on the server with optional graph node mapping (issue 2.3). */
export interface StoredSpan extends TraceEvent {
  node_id?: string | null;
}

/** Summary row for GET /api/traces. */
export interface TraceSummary {
  trace_id: string;
  span_count: number;
  started_at_us: number;
  status: SpanStatus;
  entry_kind?: EntryKind;
  entry_name?: string;
}

/** Full trace payload for GET /api/traces/:id. */
export interface TraceDetail {
  trace_id: string;
  spans: StoredSpan[];
}

export type TraceBroadcastMessage = {
  type: "span";
  trace_id: string;
  span: StoredSpan;
};

export type WsClientMessage = TraceBroadcastMessage | { type: "ready" };

const MAX_TRACES = 50;

export class TraceStore {
  private readonly traces = new Map<string, StoredSpan[]>();
  private readonly order: string[] = [];

  ingest(spans: StoredSpan[]): void {
    for (const span of spans) {
      const bucket = this.traces.get(span.trace_id);
      if (!bucket) {
        this.traces.set(span.trace_id, [span]);
        this.order.unshift(span.trace_id);
        if (this.order.length > MAX_TRACES) {
          const evicted = this.order.pop();
          if (evicted) {
            this.traces.delete(evicted);
          }
        }
        continue;
      }

      const existing = bucket.findIndex((item) => item.span_id === span.span_id);
      if (existing >= 0) {
        bucket[existing] = span;
      } else {
        bucket.push(span);
      }
    }
  }

  list(): TraceSummary[] {
    return this.order
      .map((traceId) => this.summarize(traceId))
      .filter((summary): summary is TraceSummary => summary !== null);
  }

  get(traceId: string): TraceDetail | null {
    const spans = this.traces.get(traceId);
    if (!spans?.length) {
      return null;
    }
    return {
      trace_id: traceId,
      spans: [...spans].sort((a, b) => a.start_us - b.start_us),
    };
  }

  private summarize(traceId: string): TraceSummary | null {
    const spans = this.traces.get(traceId);
    if (!spans?.length) {
      return null;
    }

    const root =
      spans.find((span) => span.parent_span_id === null || span.parent_span_id === undefined) ??
      spans[0];
    const hasError = spans.some((span) => span.status === "error");

    return {
      trace_id: traceId,
      span_count: spans.length,
      started_at_us: Math.min(...spans.map((span) => span.start_us)),
      status: hasError ? "error" : "ok",
      entry_kind: root?.entry_kind,
      entry_name: root?.entry_name,
    };
  }
}

export type TraceSubscriber = (message: TraceBroadcastMessage) => void;

export class TraceHub {
  private readonly subscribers = new Set<TraceSubscriber>();

  subscribe(listener: TraceSubscriber): () => void {
    this.subscribers.add(listener);
    return () => {
      this.subscribers.delete(listener);
    };
  }

  publish(span: StoredSpan): void {
    const message: TraceBroadcastMessage = {
      type: "span",
      trace_id: span.trace_id,
      span,
    };
    for (const listener of this.subscribers) {
      listener(message);
    }
  }
}
