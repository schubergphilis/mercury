#FROM golang:1.8
FROM scratch
ADD ./build/docker/ca-certificates.crt /etc/ssl/certs/
ADD ./build/docker/mercury-docker.toml /
ADD ./bin/linux/mercury /
ENTRYPOINT ["/mercury","-config-file","/mercury-docker.toml","-pid-file","/mercury.pid"]
