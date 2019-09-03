FROM golang:alpine as builder
RUN mkdir -p /go/src/github.com/schubergphilis/mercury
ADD . /go/src/github.com/schubergphilis/mercury/
WORKDIR /go/src/github.com/schubergphilis/mercury/
RUN apk add --no-cache git
ENV GOPATH /go/
ENV GOBIN /go/bin
RUN ls -al
RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o build/linux/mercury cmd/mercury/mercury.go
#FROM scratch
FROM golang:alpine
COPY --from=builder /go/src/github.com/schubergphilis/mercury/build/linux/mercury /app/
RUN mkdir -p /etc/mercury/ssl
COPY  test/mercury.toml.docker /etc/mercury/mercury.toml
COPY  test/ssl/self_signed_certificate.key /etc/mercury/ssl
COPY  test/ssl/self_signed_certificate.crt /etc/mercury/ssl
WORKDIR /app
EXPOSE 9000 9001 80 443
ENTRYPOINT ["/app/mercury", "--config-file", "/etc/mercury/mercury.toml"]
