.DEFAULT_GOAL := docker-image

IMAGE ?= k8snsctrl

.PHONY:build
build: 
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/admission-controller

.PHONY: docker-image
docker-image: build
	docker build -t $(IMAGE) .

.PHONY: push-image
push-image: docker-image
	docker push $(IMAGE)

.PHONY: build-k3s 
build-k3s: docker-image push-image-k3s delete-pod

# Import image to k3s
push-image-k3s:
	sudo scripts/import_image_to_k3s.sh

delete-pod:
	scripts/delete_pod.sh