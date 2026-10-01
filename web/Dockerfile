# syntax=docker/dockerfile:1.7

# Node 22 is a hard floor, not a preference. Nuxt 4.5 / Nitro 2.13 depend on
# the `trustedFunctions` global (Node 22.12+); on node:20 the build dies with
# "TypeError: trustedFunctions.difference is not a function". Keep this in sync
# with the "engines" field in package.json and the NODE_VERSION in ci.yml.

# ---- build stage ----
FROM node:22-alpine AS build
WORKDIR /app

# `npm ci` (not `npm install`): it installs exactly what the lockfile pins and
# fails loudly if package.json and package-lock.json have drifted apart.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

# .dockerignore keeps node_modules/.nuxt/.output out of the context, so this
# COPY cannot clobber the freshly installed dependency tree.
COPY . .

# `nuxt prepare` regenerates .nuxt/tsconfig.json and the #build aliases that
# `npm run build` depends on; the postinstall hook already did this, and the
# explicit call keeps the build correct even with --ignore-scripts.
RUN npm run postinstall && npm run build

# ---- runtime stage ----
FROM node:22-alpine AS runtime
WORKDIR /app

ENV NODE_ENV=production \
    PORT=3000 \
    NITRO_HOST=0.0.0.0 \
    NITRO_PORT=3000

# Only the built output ships. No node_modules, no source, no toolchain.
COPY --from=build /app/.output ./.output
COPY --from=build /app/public ./public

EXPOSE 3000

# /healthz is a Nitro route that never touches the Go backend, so a database
# blip cannot cause the frontend container to be restarted.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
    CMD wget -qO- "http://127.0.0.1:${PORT}/healthz" >/dev/null || exit 1

# `node` runs unprivileged by default in the official image (uid 1000).
CMD ["node", ".output/server/index.mjs"]
