import type { FastifyInstance } from "fastify";
import type { TraceHub } from "../trace/store.js";

export async function registerWebSocketRoutes(app: FastifyInstance, hub: TraceHub): Promise<void> {
  app.get("/ws", { websocket: true }, (socket) => {
    socket.send(JSON.stringify({ type: "ready" }));

    const unsubscribe = hub.subscribe((message) => {
      if (socket.readyState === socket.OPEN) {
        socket.send(JSON.stringify(message));
      }
    });

    socket.on("close", () => {
      unsubscribe();
    });
  });
}
