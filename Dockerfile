FROM golang:latest

WORKDIR /go/src/github.com/onestay/resize-server
COPY . .

RUN go get -d -v ./...
RUN go build

EXPOSE 3001

CMD ["go", "run"]

