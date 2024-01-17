#Stage 1
FROM golang:1.21.4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o indexer .

# Stage 2
FROM alpine:latest
RUN apk --no-cache add ca-certificates bash
WORKDIR /root/
COPY --from=builder /app/indexer /usr/bin
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["indexer", "start", "--block", "8713586" , "--api"]
