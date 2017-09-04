FROM golang:alpine

WORKDIR /go/src/github.com/onestay/resize-server
COPY . .

RUN apk update \
	&& apk add git

RUN go get -d -v ./...
RUN go build

EXPOSE 3001

CMD ["go", "run"]

