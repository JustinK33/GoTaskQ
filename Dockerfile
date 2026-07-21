FROM golang:1.23-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/conduit ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /out/conduit /conduit

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/conduit"]
