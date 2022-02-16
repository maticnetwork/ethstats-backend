FROM golang:1.17-alpine as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./

RUN go build -o /wsimple

EXPOSE 8000

FROM alpine:3.15
COPY --from=builder /wsimple .

# executable
ENTRYPOINT [ "./wsimple server" ]
