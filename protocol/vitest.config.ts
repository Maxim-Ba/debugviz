import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    // WSL + /mnt/u: esbuild worker often crashes with default thread pool.
    pool: "forks",
    fileParallelism: false,
    deps: {
      optimizer: {
        ssr: {
          enabled: false,
        },
      },
    },
  },
});
