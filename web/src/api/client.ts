import type { Graph, GraphMeta } from "@debugviz/protocol";

const API_BASE = import.meta.env.VITE_API_URL ?? "";

export interface StoredSpan {
  trace_id: string;
  span_id: string;
  parent_span_id?: string | null;
  name: string;
  file: string;
  line: number;
  start_us: number;
  duration_us: number;
  status: "ok" | "error";
  entry_kind?: string;
  entry_name?: string;
  node_id?: string | null;
}

export interface TraceSummary {
  trace_id: string;
  span_count: number;
  started_at_us: number;
  status: "ok" | "error";
  entry_kind?: string;
  entry_name?: string;
}

export interface TraceDetail {
  trace_id: string;
  spans: StoredSpan[];
}

export type WsMessage =
  | { type: "ready" }
  | { type: "span"; trace_id: string; span: StoredSpan };

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, init);
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`${response.status} ${path}: ${body || response.statusText}`);
  }
  return response.json() as Promise<T>;
}

export function fetchGraph(): Promise<Graph> {
  return apiFetch<Graph>("/api/graph");
}

export function fetchGraphMeta(): Promise<GraphMeta> {
  return apiFetch<GraphMeta>("/api/graph/meta");
}

export function fetchTraces(): Promise<TraceSummary[]> {
  return apiFetch<TraceSummary[]>("/api/traces");
}

export function fetchTrace(traceId: string): Promise<TraceDetail> {
  return apiFetch<TraceDetail>(`/api/traces/${encodeURIComponent(traceId)}`);
}

export function wsUrl(): string {
  const apiBase = import.meta.env.VITE_API_URL as string | undefined;
  if (apiBase) {
    const parsed = new URL(apiBase);
    const protocol = parsed.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${parsed.host}/ws`;
  }
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws`;
}
