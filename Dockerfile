FROM golang:1.17-alpine as builder

WORKDIR /app

COPY ["go.mod", "go.sum", "*.go", "./"]

RUN set -x \ 
  && go mod download \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dsmr-adapter

FROM scratch

COPY --from=builder /dsmr-adapter /usr/bin/

CMD ["dsmr-adapter"]