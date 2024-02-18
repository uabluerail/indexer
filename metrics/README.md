# To start prometheus + grafana

`cd metrics`

`docker compose up -d --build`

### Note: remember to allow ports for Prometheus to see host.docker.internal:xxxx from within container

Lister, consumer, indexer
`sudo ufw allow 11001`
`sudo ufw allow 11002`
`sudo ufw allow 11003`

Postgres
`sudo ufw allow 15432`

# Go to `metrics/prometheus/exporters` and install node and query exporters