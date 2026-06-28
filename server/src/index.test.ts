import { describe, expect, it } from "vitest";
import { buildServer } from "./app.js";

describe("buildServer", () => {
  it("responds on /health", async () => {
    const app = await buildServer();
    const response = await app.inject({ method: "GET", url: "/health" });
    expect(response.statusCode).toBe(200);
    expect(response.json()).toEqual({ status: "ok" });
    await app.close();
  });
});
