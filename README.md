
# Ethstats-server

Start websocket server:

```
$ export DBNAME=<Database Name>
$ export DBPASS=<Database password>
$ go run .
```

Start geth client:

```
$ docker run --net=host ethereum/client-go --dev --ethstats a:b@localhost:8080
```
