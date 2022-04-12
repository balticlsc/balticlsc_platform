FROM ubuntu:18.04

ENV RANCHER_URL=https://10.0.132.151
ENV RANCHER_TOKEN=token-l6kbh:cxfq965rtwfn4dxjjghdff5bsqh9ln79ntl5djb8j52xkljnn875gp
ENV RANCHER_CLUSTER_ID=c-jb5zn

COPY ./bin /bin
ENTRYPOINT ["/bin/admission-controller"]
