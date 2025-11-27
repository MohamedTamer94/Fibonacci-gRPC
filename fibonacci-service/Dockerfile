FROM golang:1.25-alpine

WORKDIR /app

COPY ./fibonacci-service/go.* ./
COPY proto/go.* ./proto/

RUN go mod download

COPY ./fibonacci-service/ .
COPY proto/ ./proto/

RUN go build -o fibonacci-service main.go

CMD ["./fibonacci-service"]