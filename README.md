# DSMR Prometheus Adapter

This project will run a webserver that serves statistics on it's /metrics endpoint to be scraped by prometheus. It will use the `DSMR_BASE_URL` environement variable to connect to the [DSMR-logger](https://opencircuit.shop/blog/DSMR-logger-V4-Slimme-Meter-uitlezer) of your choice.

## Building

```shell
docker build -t dsmr-adapter .
```

## Disclaimer

This is my first Go program ever it probably sucks.