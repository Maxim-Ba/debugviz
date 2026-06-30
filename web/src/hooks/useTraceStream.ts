import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { StoredSpan, TraceSummary, WsMessage } from "../api/client.js";
import { fetchTrace, fetchTraces, wsUrl } from "../api/client.js";

export interface TraceState {
  summaries: TraceSummary[];
  activeTraceId: string | null;
  spans: StoredSpan[];
  highlightNodeIds: Set<string>;
  errorNodeIds: Set<string>;
  connected: boolean;
}

function buildPathNodeIds(spans: StoredSpan[]): { highlights: Set<string>; errors: Set<string> } {
  const highlights = new Set<string>();
  const errors = new Set<string>();
  for (const span of spans) {
    if (span.node_id) {
      highlights.add(span.node_id);
      if (span.status === "error") {
        errors.add(span.node_id);
      }
    }
  }
  return { highlights, errors };
}

export function useTraceStream(enabled: boolean) {
  const [summaries, setSummaries] = useState<TraceSummary[]>([]);
  const [activeTraceId, setActiveTraceId] = useState<string | null>(null);
  const [spans, setSpans] = useState<StoredSpan[]>([]);
  const [connected, setConnected] = useState(false);
  const spansByTrace = useRef(new Map<string, Map<string, StoredSpan>>());

  const refreshSummaries = useCallback(async () => {
    try {
      const list = await fetchTraces();
      setSummaries(list);
    } catch {
      // server may have no traces yet
    }
  }, []);

  const selectTrace = useCallback(async (traceId: string) => {
    setActiveTraceId(traceId);
    try {
      const detail = await fetchTrace(traceId);
      setSpans(detail.spans);
    } catch {
      setSpans([]);
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void refreshSummaries();

    const ws = new WebSocket(wsUrl());
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    ws.onmessage = (event) => {
      const msg = JSON.parse(String(event.data)) as WsMessage;
      if (msg.type !== "span") {
        return;
      }

      const bucket = spansByTrace.current.get(msg.trace_id) ?? new Map<string, StoredSpan>();
      bucket.set(msg.span.span_id, msg.span);
      spansByTrace.current.set(msg.trace_id, bucket);

      setActiveTraceId(msg.trace_id);
      setSpans([...bucket.values()].sort((a, b) => a.start_us - b.start_us));
      void refreshSummaries();
    };

    return () => {
      ws.close();
      setConnected(false);
    };
  }, [enabled, refreshSummaries]);

  const path = useMemo(() => buildPathNodeIds(spans), [spans]);

  return {
    summaries,
    activeTraceId,
    spans,
    connected,
    highlightNodeIds: path.highlights,
    errorNodeIds: path.errors,
    selectTrace,
    refreshSummaries,
  };
}
