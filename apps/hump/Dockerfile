# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e
# Frontend docker/dockerfile 1.27.1.

# Build context: the repo root (docker build -f apps/hump/Dockerfile .).
# The bun and Go versions must match .moon/toolchains.yml; Dependabot does not update that file.

# hump-tower UI. Runs on the build platform: the output is the same for every target.
FROM --platform=$BUILDPLATFORM oven/bun:1.4.3@sha256:ec06c3b6cea04192ae6770c434f668ca41d343ad19fa6472216c7b48be39c598 AS ui

# A fixed path for the cache mount below.
ENV BUN_INSTALL_CACHE_DIR=/root/.bun/install/cache

WORKDIR /src

# Manifests first, so source changes do not invalidate the install layer.
# No NODE_ENV=production here: the build needs the devDependencies (rsbuild, openapi-ts).
COPY package.json bun.lock bunfig.toml ./
COPY apps/hump-tower/package.json apps/hump-tower/bunfig.toml ./apps/hump-tower/
RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile

COPY apps/hump-tower ./apps/hump-tower

ARG VERSION=0.0.0

# rsbuild writes to ../hump/internal/controller/hump_tower, outside the package.
RUN mkdir -p /src/apps/hump/internal/controller \
    && cd apps/hump-tower \
    && NODE_ENV=production VERSION=$VERSION bun run build

# hump binary. Cross-compiles on the build platform instead of emulating the target.
FROM --platform=$BUILDPLATFORM golang:1.27.2@sha256:820153096fda4de6ac1b7194d12e416b7dd8f4e9a33d17e0345b111b945162d8 AS build

ARG TARGETOS TARGETARCH

# go.work is not in the context. GOTOOLCHAIN=local fails the build on a go.mod that needs
# a newer Go instead of downloading one.
ENV GOWORK=off \
    GOTOOLCHAIN=local \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH

WORKDIR /src/apps/hump

COPY apps/hump/go.mod apps/hump/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY apps/hump ./
# The UI built above; .dockerignore keeps the committed build out of the context.
COPY --from=ui /src/apps/hump/internal/controller/hump_tower ./internal/controller/hump_tower

# Passed in instead of scripts/ldflags.sh: there is no .git in the context to read the commit from.
ARG VERSION=0.0.0 COMMIT=unknown DATE=

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath \
    -ldflags "-s -w -X hump/pkg/version.Version=${VERSION} -X hump/pkg/version.Commit=${COMMIT} -X hump/pkg/version.Date=${DATE}" \
    -o /out/hump ./cmd/hump

# Runtime: the static binary only; no shell, runs as nonroot.
# Digest pushed 2026-10-24: younger than the 7-day minimum age by an explicit exception.
FROM docker.io/chainguard/static:latest@sha256:c12f0107f8afb0bbc1b0eff96ff8675e85a10c6083d039b777c51fef5c27420f

COPY --from=build /out/hump /usr/bin/hump

# nonroot is the image default; set explicitly.
USER 65532:65532

# Web UI/API and Prometheus metrics. Both stay off unless WEB_PORT / TELEMETRY_PORT are set.
EXPOSE 3200 9100

ENTRYPOINT ["/usr/bin/hump"]
