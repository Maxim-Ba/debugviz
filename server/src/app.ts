import Fastify from "fastify";
import websocket from "@fastify/websocket";

export async function buildServer() {
  const app = Fastify({ logger: false });

  await app.register(websocket);

  app.get("/health", async () => ({ status: "ok" }));

  app.get("/api/ws", { websocket: true }, (socket) => {
    socket.send(JSON.stringify({ type: "ready" }));
    socket.on("message", (message: Buffer | ArrayBuffer | Buffer[]) => {
      socket.send(message);
    });
  });

  return app;
}
