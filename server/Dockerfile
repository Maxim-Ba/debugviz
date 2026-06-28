FROM node:22-alpine AS builder

WORKDIR /app
RUN corepack enable

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY server/package.json ./server/
RUN pnpm install --frozen-lockfile

COPY server ./server
COPY eslint.config.js ./
RUN pnpm --filter @debugviz/server build

FROM node:22-alpine
WORKDIR /app
RUN corepack enable

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY server/package.json ./server/
RUN pnpm install --frozen-lockfile --prod

COPY --from=builder /app/server/dist ./server/dist
COPY server/package.json ./server/

WORKDIR /app/server
ENV HOST=0.0.0.0
ENV PORT=4000
EXPOSE 4000
CMD ["node", "dist/index.js"]
