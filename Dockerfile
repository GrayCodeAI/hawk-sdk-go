FROM golang:1.26.4-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY go.mod ./
# go.sum only exists if there are external dependencies
COPY go.sum* ./
RUN go mod download && go mod verify

COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w \
      -X main.Version=${VERSION} \
      -X main.Commit=${COMMIT} \
      -X main.BuildDate=${BUILD_DATE}" \
    -o hawk-sdk-go-example .  # SDK has no main; build verification only

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tini && \
    adduser -D -u 1000 hawk

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER hawk
WORKDIR /workspace
ENTRYPOINT ["tini", "--"]
CMD ["sleep", "infinity"]
