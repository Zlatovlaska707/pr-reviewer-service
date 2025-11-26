FROM golang:1.24.10 AS build
WORKDIR /app

# Настраиваем Go proxy с fallback на прямой доступ
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download || \
    (sleep 2 && go mod download) || \
    (sleep 5 && go mod download)

COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CLEAN_CACHE=false
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -o /build/pr-reviewer ./cmd/run

RUN if [ "$CLEAN_CACHE" = "true" ]; then go clean -cache -modcache; fi

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /srv
COPY --from=build /build/pr-reviewer /srv/pr-reviewer
COPY --from=build /app/migrations /srv/migrations
COPY --from=build /app/openapi.yml /srv/openapi.yml
COPY --from=build /app/config/config.yaml /srv/config.yaml
ENV MIGRATIONS_PATH=/srv/migrations
ENV SWAGGER_SPEC_PATH=/srv/openapi.yml
ENV CONFIG_PATH=/srv/config.yaml
EXPOSE 8080
ENTRYPOINT ["/srv/pr-reviewer"]

