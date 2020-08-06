FROM golang:1.13-alpine3.10

RUN GO111MODULE=on go get github.com/cortesi/modd/cmd/modd

RUN apk add --no-cache build-base

WORKDIR /usr/src/api

COPY . .

RUN go mod download

EXPOSE 80

ENTRYPOINT [ "modd" ]