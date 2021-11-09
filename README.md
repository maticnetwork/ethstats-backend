
# Ethstats-server

Start websocket server:

```
$ go run main.go
```

Start geth client:

```
$ docker run --net=host ethereum/client-go --dev --ethstats a:b@localhost:8080
```
