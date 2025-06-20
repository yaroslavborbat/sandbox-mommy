FROM docker.io/golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY api/ api/

RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o sandbox-api cmd/apiserver/main.go

FROM ubuntu:22.04
WORKDIR /app
COPY --from=builder /workspace/sandbox-api /app/sandbox-api
COPY --from=builder /go/bin/dlv /app/dlv
USER 65532:65532

ENTRYPOINT ["./dlv", "--listen=:2345", "--headless=true", "--continue", "--log=true", "--log-output=debugger,debuglineerr,gdbwire,lldbout,rpc", "--accept-multiclient", "--api-version=2", "exec", "/app/sandbox-api", "--"]
