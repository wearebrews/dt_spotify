FROM golang:1.15 as base

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/dtspotify/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=base /app/main ./main

EXPOSE 8080

CMD ["./main"]
