
# Ethstats-server

## Development

Run postgresql (user=postgresql, pass=postgresql):

```
$ make postgresql-test
```

Run postgresql admin panel (email=postgres@gmail.com, pass=postgres, http=localhost:80):

```
$ postgresql-test-admin
```

Run ethstats backend:

```
$ go run main.go
```

Start geth client:

```
$ docker run --net=host ethereum/client-go --dev --ethstats a:b@localhost:8080
```
