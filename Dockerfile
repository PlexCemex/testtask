FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /server /app/server
COPY config.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/server"]
