FROM golang:1.14

WORKDIR /go/src/consumer
COPY consumer.go .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["consumer"]
