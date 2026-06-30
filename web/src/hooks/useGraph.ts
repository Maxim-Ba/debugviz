import { useCallback, useEffect, useState } from "react";
import type { Graph, GraphMeta } from "@debugviz/protocol";
import { fetchGraph, fetchGraphMeta } from "../api/client.js";

export function useGraph(pollMs = 5000) {
  const [graph, setGraph] = useState<Graph | null>(null);
  const [meta, setMeta] = useState<GraphMeta | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    try {
      const [g, m] = await Promise.all([fetchGraph(), fetchGraphMeta()]);
      setGraph(g);
      setMeta(m);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
    const timer = window.setInterval(() => {
      void reload();
    }, pollMs);
    return () => window.clearInterval(timer);
  }, [pollMs, reload]);

  return { graph, meta, error, loading, reload };
}
