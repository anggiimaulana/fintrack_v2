# ============================================================
# Frontend — Next.js (standalone-ish, pnpm)
# ============================================================
FROM node:22-alpine

WORKDIR /app

# Install pnpm 11 to match the committed lockfile + workspace build policy.
RUN corepack enable pnpm && corepack prepare pnpm@11 --activate

# Copy package management files.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

RUN pnpm install --frozen-lockfile

COPY . .

# Build-time env: same-origin API path + backend origin for the rewrite proxy.
ARG BACKEND_ORIGIN=http://backend:8081
ENV BACKEND_ORIGIN=${BACKEND_ORIGIN}
ENV NEXT_PUBLIC_API_URL=/api/v1
ENV CI=true
RUN pnpm build

ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

EXPOSE 3000

CMD ["pnpm", "start"]
