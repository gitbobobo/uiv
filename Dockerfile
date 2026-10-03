# syntax=docker/dockerfile:1

# Build stages run on the builder's native platform; only the Go binary is cross-compiled.
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src
RUN npm install -g pnpm@11.0.8
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json web/
RUN pnpm install --frozen-lockfile
COPY web/ web/
RUN pnpm --filter web build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
ARG TARGETOS TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/uiv ./cmd/uiv

FROM gcr.io/distroless/static-debian12
COPY --from=go /out/uiv /uiv
ENV UIV_DATA_DIR=/data UIV_ADDR=:8080
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/uiv"]
