module cluster-proxy

go 1.13

require (
	github.com/golang/protobuf v1.4.2
	github.com/sirupsen/logrus v1.7.0
	github.com/tidwall/gjson v1.6.1
	google.golang.org/grpc v1.32.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.0.1 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/resty.v1 v1.12.0
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
)
