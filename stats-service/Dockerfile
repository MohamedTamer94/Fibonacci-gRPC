FROM golang:1.25-alpine

WORKDIR /app

COPY ./stats-service/go.* ./
COPY proto/go.* ./proto/

RUN go mod download

COPY ./stats-service/ .
COPY proto/ ./proto/

RUN go build -o stats-service main.go

CMD ["./stats-service"]