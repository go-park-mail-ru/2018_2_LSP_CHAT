FROM golang:alpine

RUN apk add --no-cache git

ADD . /go/src/github.com/go-park-mail-ru/2018_2_LSP_CHAT

RUN cd /go/src/github.com/go-park-mail-ru/2018_2_LSP_CHAT && go get ./...

RUN go install github.com/go-park-mail-ru/2018_2_LSP_CHAT

ENTRYPOINT /go/bin/2018_2_LSP_CHAT

EXPOSE 8080