ARG GO_VERSION=1.27.2

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /builder

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -tags netgo -trimpath -ldflags="-s -w -extldflags '-static'" -o service ./cmd/service


FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /builder/service .

EXPOSE 8080

USER nonroot

ENTRYPOINT [ "./service" ]
