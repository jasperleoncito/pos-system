# ---- base: shared Go toolchain layer ----
FROM golang:1.25-bookworm AS base
WORKDIR /app
ENV GOCACHE=/go-cache GOMODCACHE=/gomod-cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/gomod-cache go mod download

# ---- dev: hot reload via air (compose bind-mounts the source) ----
FROM base AS dev
RUN go install github.com/air-verse/air@v1.61.7
COPY . .
CMD ["air", "-c", ".air.toml"]

# ---- build: compile static binaries ----
FROM base AS build
COPY . .
RUN --mount=type=cache,target=/go-cache --mount=type=cache,target=/gomod-cache \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- prod: minimal runtime ----
FROM gcr.io/distroless/base-debian12 AS prod
WORKDIR /app
COPY --from=build /out/api /out/worker /out/seed /app/
COPY migrations /app/migrations
EXPOSE 9137
CMD ["/app/api"]
