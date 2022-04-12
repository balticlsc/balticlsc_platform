IMAGENAME = danielxolsson/cluster-proxy:latest

all: grpc server client

.PHONY: server client

server:
	go build -o bin/server server/main.go
	
client:
	go build -o bin/client client/main.go

grpc:
	protoc -I proto/ proto/clusterproxy.proto --go_out=. --go-grpc_out=. 

dockerimage:
	docker build --tag $(IMAGENAME) .

dockerpush:
	docker push $(IMAGENAME)
	
