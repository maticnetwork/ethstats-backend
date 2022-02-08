# syntax=docker/dockerfile:1

FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /wsimple

EXPOSE 8000

#Persist data for these days. Deletes older data.
ENV PERSIST_DAYS 5
CMD [ "sh", "-c",  "/wsimple --collector.secret hello --persist-days ${PERSIST_DAYS}" ]