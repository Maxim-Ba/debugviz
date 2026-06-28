import { buildServer } from "./app.js";

const host = process.env.HOST ?? "0.0.0.0";
const port = Number(process.env.PORT ?? 4000);

async function main() {
  const app = await buildServer();
  await app.listen({ host, port });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
