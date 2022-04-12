FROM ubuntu:18.04

ENV RANCHER_URL=https://k8s.ice.ri.se
ENV RANCHER_CLUSTER_ID=c-tmfxj
ENV RANCHER_PROJECT_ID=p-vxl6k
ENV RANCHER_TOKEN=token-l6kbh:cxfq965rtwfn4dxjjghdff5bsqh9ln79ntl5djb8j52xkljnn875gp
ENV TLS_INSECURE=true
ENV LISTEN_PORT=50051

COPY ./bin /bin
ENTRYPOINT ["/bin/server"]
