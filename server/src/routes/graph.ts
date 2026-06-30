import type { FastifyInstance } from "fastify";
import { computeGraphMeta } from "@debugviz/protocol";
import type { AppState } from "../state.js";
import { setGraph } from "../state.js";
import { ValidationError, assertValidGraph } from "../validate.js";

export async function registerGraphRoutes(app: FastifyInstance, state: AppState): Promise<void> {
  app.post("/api/graph", async (request, reply) => {
    try {
      state.graph = assertValidGraph(request.body);
      setGraph(state, state.graph);
      return reply.code(201).send({ ok: true });
    } catch (err) {
      if (err instanceof ValidationError) {
        return reply.code(400).send({ error: err.message });
      }
      throw err;
    }
  });

  app.get("/api/graph", async (_request, reply) => {
    if (!state.graph) {
      return reply.code(404).send({ error: "graph not loaded" });
    }
    return state.graph;
  });

  app.get("/api/graph/meta", async (_request, reply) => {
    if (!state.graph) {
      return reply.code(404).send({ error: "graph not loaded" });
    }
    return computeGraphMeta(state.graph);
  });
}
