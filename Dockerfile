## Build binary
FROM golang:1.24 as builder
WORKDIR /app
COPY ./ ./

RUN CGO_ENABLED=0 go build -o ./freyr -ldflags "-extldflags '-static'" cmd/main.go
RUN chmod 777 ./freyr

## Output Runner
FROM alpine
WORKDIR /app
COPY --from=builder /app/freyr /app/freyr
ENTRYPOINT /app/freyr
