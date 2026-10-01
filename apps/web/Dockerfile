# Build context is the repo root so pnpm can use the workspace lockfile.
FROM node:26-alpine AS base
RUN npm install -g pnpm@12.3.4

FROM base AS build
WORKDIR /repo
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml ./
COPY apps/web/package.json apps/web/
RUN pnpm install --frozen-lockfile --filter web
COPY apps/web apps/web
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN pnpm --filter web build

FROM base AS run
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /repo/apps/web/.next/standalone ./
COPY --from=build /repo/apps/web/.next/static apps/web/.next/static
COPY --from=build /repo/apps/web/public apps/web/public
EXPOSE 3000
CMD ["node", "apps/web/server.js"]
