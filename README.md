
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
$ go run main.go --collector.secret secret [--frontend.addr ws://localhost:3000/api --frontend.secret secret2]
```

Start geth client:

```
$ docker run --net=host ethereum/client-go --dev --dev.period 1 --ethstats a:secret@localhost:8000
```

Run ethstats frontend (goerli/ethstats repo):

```
$ grunt poa
$ WS_SECRET="secret2" npm start
```

## Flags

- db-endpoint: Database endpoint to store the data.

- collector.addr (default=localhost:8000): Websocket address to collect metrics.

- collector.secret (default=''): Secret for the local websocket collector.

- log-level (default=info): Level to log the output.

- frontend.addr: Address of the ethstats frontend to proxy the data.

- frontend.secret: Secret to be used in the ethstats proxy.
