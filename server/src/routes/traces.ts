import type { FastifyInstance } from "fastify";
import type { TraceEvent } from "@debugviz/protocol";
import type { AppState } from "../state.js";
import { ValidationError, assertValidTraceEvent } from "../validate.js";
import type { StoredSpan, TraceHub, TraceStore } from "../trace/store.js";

function parseSpanBatch(body: unknown): unknown[] {
  if (Array.isArray(body)) {
    return body;
  }
  if (body && typeof body === "object" && Array.isArray((body as { spans?: unknown }).spans)) {
    return (body as { spans: unknown[] }).spans;
  }
  throw new ValidationError("expected { spans: [...] } or a spans array");
}

function toStoredSpan(event: TraceEvent, state: AppState): StoredSpan {
  const node_id = state.mapper?.map(event) ?? null;
  return { ...event, node_id };
}

export async function registerTraceRoutes(
  app: FastifyInstance,
  state: AppState,
  traces: TraceStore,
  hub: TraceHub,
): Promise<void> {
  app.post("/api/traces/spans", async (request, reply) => {
    try {
      const rawSpans = parseSpanBatch(request.body);
      const stored: StoredSpan[] = [];

      for (const raw of rawSpans) {
        const event = assertValidTraceEvent(raw);
        const span = toStoredSpan(event, state);
        stored.push(span);
      }

      traces.ingest(stored);
      for (const span of stored) {
        hub.publish(span);
      }

      return reply.code(202).send({ accepted: stored.length });
    } catch (err) {
      if (err instanceof ValidationError) {
        return reply.code(400).send({ error: err.message });
      }
      throw err;
    }
  });

  app.get("/api/traces", async () => traces.list());

  app.get<{ Params: { id: string } }>("/api/traces/:id", async (request, reply) => {
    const detail = traces.get(request.params.id);
    if (!detail) {
      return reply.code(404).send({ error: "trace not found" });
    }
    return detail;
  });
}
