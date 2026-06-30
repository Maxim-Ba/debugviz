import Fastify from "fastify";
import websocket from "@fastify/websocket";
import { registerGraphRoutes } from "./routes/graph.js";
import { registerTraceRoutes } from "./routes/traces.js";
import { registerWebSocketRoutes } from "./routes/ws.js";
import { createAppState } from "./state.js";
import { TraceHub, TraceStore } from "./trace/store.js";

export async function buildServer() {
  const app = Fastify({ logger: false });
  const state = createAppState();
  const traces = new TraceStore();
  const hub = new TraceHub();

  await app.register(websocket);

  app.get("/health", async () => ({ status: "ok" }));

  await registerGraphRoutes(app, state);
  await registerTraceRoutes(app, state, traces, hub);
  await registerWebSocketRoutes(app, hub);

  return app;
}
