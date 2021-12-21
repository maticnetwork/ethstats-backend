
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
$ go run main.go [--frontend-addr ws://localhost:3000/api]
```

Start geth client:

```
$ docker run --net=host ethereum/client-go --dev --dev.period 1 --ethstats a:b@localhost:8000
```

Run ethstats frontend (goerli/ethstats repo):

```
$ grunt poa
$ WS_SECRET="b" npm start
```
