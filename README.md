
# Ethstats-server

## Development

Run postgresql (user=postgresql(default), pass=postgrespassword):

```
$ make postgresql-test
```

Run postgresql admin panel (email=postgres@gmail.com, pass=postgrespassword, http=localhost:80):

```
$ make postgresql-test-admin
```

Run ethstats backend:

```
$ go run main.go server \
    --collector.secret secret \
    --db-endpoint "postgres://postgres:postgrespassword@127.0.0.1:5432/postgres?sslmode=disable" \
    --frontend.addr ws://localhost:3000/api \
    --frontend.secret secret2
```

Start geth client:

```
$ geth --dev --dev.period 1 --ethstats a:secret@localhost:8000
```

Run ethstats frontend ([goerli/ethstats-server repo](https://github.com/goerli/ethstats-server)):

```
$ npm ci
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

- save-block-txs: Whether block transactions should be written to database.
