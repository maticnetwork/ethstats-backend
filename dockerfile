# syntax=docker/dockerfile:1

FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /wsimple

EXPOSE 8000

CMD [ "/wsimple", "--collector.secret", "hello" ]