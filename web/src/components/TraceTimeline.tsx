import type { StoredSpan } from "../api/client.js";

interface TraceTimelineProps {
  spans: StoredSpan[];
  activeTraceId: string | null;
}

function formatDuration(us: number): string {
  if (us >= 1_000_000) {
    return `${(us / 1_000_000).toFixed(2)}s`;
  }
  if (us >= 1000) {
    return `${(us / 1000).toFixed(1)}ms`;
  }
  return `${us}µs`;
}

export function TraceTimeline({ spans, activeTraceId }: TraceTimelineProps) {
  if (!spans.length) {
    return (
      <div className="trace-panel empty">
        <h2>Live trace</h2>
        <p>Ожидание spans по WebSocket…</p>
      </div>
    );
  }

  const t0 = Math.min(...spans.map((s) => s.start_us));
  const total = Math.max(...spans.map((s) => s.start_us + s.duration_us)) - t0;

  return (
    <div className="trace-panel">
      <h2>Trace {activeTraceId}</h2>
      <ul className="timeline">
        {spans.map((span) => {
          const offset = span.start_us - t0;
          const widthPct = total > 0 ? (span.duration_us / total) * 100 : 10;
          const leftPct = total > 0 ? (offset / total) * 100 : 0;
          return (
            <li key={span.span_id} className={span.status === "error" ? "error" : ""}>
              <div className="span-label">
                <span className="name">{span.name}</span>
                <span className="meta">
                  {formatDuration(span.duration_us)}
                  {span.node_id ? ` · ${span.node_id}` : ""}
                </span>
              </div>
              <div className="span-bar-track">
                <div
                  className="span-bar"
                  style={{ left: `${leftPct}%`, width: `${Math.max(widthPct, 2)}%` }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
