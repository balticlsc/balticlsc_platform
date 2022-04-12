FROM golang:1.14

WORKDIR /go/src/producer
COPY producer.go .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["producer"]
