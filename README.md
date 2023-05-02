
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


## Run local docker compose environment
- ``` git clone https://github.com/maticnetwork/reorgs-frontend.git```
- ```cd reorgs-frontend```
- ```git checkout localhost```
- ```sudo docker build -t ethstats-frontend .```
- ```cd ..```
- ```sudo docker build -t ethstats-backend .```
- ```docker-compose up -d```
- go to ```localhost:8080``` to see the hasura frontend
- click on ```settings``` on top-right corner
- click on ```import metadata```
- select ```hasura_metadata_example.json``` from the root directory of this repo
- go to ```localhost:3000``` to see the frontend

### Add ethstats flag in bor commands to send bor data to ethstats-backend
- ```--ethstats <node-name>:<secret>@<ethstats-server-ip>:<ethstats-server-port>```
- for local setup : ```--ethstats node1:hello@localhost:8000```
