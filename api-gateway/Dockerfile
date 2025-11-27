FROM golang:1.25-alpine

WORKDIR /app

COPY ./api-gateway/go.* ./
COPY proto/go.* ./proto/

RUN go mod download

COPY ./api-gateway/ .
COPY proto/ ./proto/

RUN go build -o api-gateway main.go

CMD ["./api-gateway"]