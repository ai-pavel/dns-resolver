FROM golang:1.21 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o resolve ./cmd/resolve

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /app/resolve /usr/local/bin/resolve
EXPOSE 53/udp
EXPOSE 53/tcp
ENTRYPOINT ["/usr/local/bin/resolve"]