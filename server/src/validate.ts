import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import Ajv2020, { type ErrorObject, type ValidateFunction } from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import type { Graph, TraceEvent } from "@debugviz/protocol";

const repoRoot = join(dirname(fileURLToPath(import.meta.url)), "..", "..");

function loadSchema(name: string): object {
  const path = join(repoRoot, "schemas", name);
  return JSON.parse(readFileSync(path, "utf8")) as object;
}

function createValidator<T>(schema: object): ValidateFunction<T> {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  addFormats(ajv);
  return ajv.compile<T>(schema);
}

const validateGraphDoc = createValidator<Graph>(loadSchema("graph.schema.json"));
const validateTraceEvent = createValidator<TraceEvent>(loadSchema("trace-event.schema.json"));

export function formatValidationErrors(errors: ErrorObject[] | null | undefined): string {
  if (!errors?.length) {
    return "validation failed";
  }
  return errors.map((err) => `${err.instancePath || "/"} ${err.message ?? ""}`.trim()).join("; ");
}

export function assertValidGraph(graph: unknown): Graph {
  if (!validateGraphDoc(graph)) {
    throw new ValidationError(formatValidationErrors(validateGraphDoc.errors));
  }
  return graph;
}

export function assertValidTraceEvent(event: unknown): TraceEvent {
  if (!validateTraceEvent(event)) {
    throw new ValidationError(formatValidationErrors(validateTraceEvent.errors));
  }
  return event;
}

export class ValidationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ValidationError";
  }
}
