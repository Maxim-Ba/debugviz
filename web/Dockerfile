FROM node:22-alpine AS builder

WORKDIR /app
RUN corepack enable

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY web/package.json ./web/
COPY protocol/package.json ./protocol/
RUN pnpm install --frozen-lockfile

COPY protocol ./protocol
COPY web ./web
COPY eslint.config.js ./
RUN pnpm --filter @debugviz/protocol build
RUN pnpm --filter @debugviz/web build

FROM nginx:1.27-alpine
COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/web/dist /usr/share/nginx/html
COPY landing /usr/share/nginx/html/landing
EXPOSE 3000
