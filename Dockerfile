FROM golang:1.16
ADD . /go/src/github.com/vmatekole/captainhook
WORKDIR /go/src/github.com/vmatekole/captainhook
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o captainhook .

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
WORKDIR /config
VOLUME /config
COPY --from=0 /go/src/github.com/vmatekole/captainhook/captainhook /usr/local/bin
ENTRYPOINT ["captainhook", "--listen-addr", "0.0.0.0:8080", "-configdir", "/config"]