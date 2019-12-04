FROM golang:1.13.4-alpine

WORKDIR /go/src/github.com/jmhodges/ensure-latest-go
COPY ./vendor ./vendor
COPY ./latest_go_ensurer ./latest_go_ensurer
COPY ./entrypoint.sh /entrypoint.sh

RUN go install github.com/jmhodges/ensure-latest-go/latest_go_ensurer

ENTRYPOINT ["/entrypoint.sh"]
