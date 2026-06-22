FROM golang:1.21-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
COPY . .
RUN go mod tidy
RUN go build -o app .

FROM alpine:latest
COPY --from=build /app/app /app
ENTRYPOINT ["/app"]
