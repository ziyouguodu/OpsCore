FROM golang:1.24-alpine AS build

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/opscore-api ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache curl
RUN adduser -D -H opscore
USER opscore
WORKDIR /app
COPY --from=build /out/opscore-api /app/opscore-api
EXPOSE 8080
CMD ["/app/opscore-api"]
