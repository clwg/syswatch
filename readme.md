# Syswatch

Syswatch strives to be a scalable and endpoint messaging reference implemenation designed to facilitate the real-time monitoring and direct command invocation across a variety of endpoints and platforms. Syswatch leverages gRPC for communication and supports both direct and broadcast messaging to endpoints and accessiable through a HTTP API.

## Status

This project is currently in alpha and should not be used in production unless you are willing to accept the risks associated with alpha software.  The project is under development and is subject to change.

## Running

```shell
git clone https://github.com/clwg/syswatch.git
cd syswatch
```

1. Generate Certificates (if needed)

If required, generate self-signed certificates for secure communication between the server and clients. Note there is no logic implemented to deal with key rotation or certificate expiry.

```shell
cd data/x509
sh create.sh
```
2. Start the Server

```shell
go run cmd/syswatch-server/syswatch-server.go -cert_file data/x509/server_cert.pem -key_file data/x509/server_key.pem -tls
```

3. Attach a Client

```shell
go run cmd/syswatch-client/syswatch-client.go -addr localhost:51001 -ca_file data/x509/ca_cert.pem -tls -filelist filelist.txt
```

### API

#### List Connections

```shell
 curl -X GET http://localhost:8084/connections
 ```

#### Send Commands

```shell
curl -X POST -H "Content-Type: application/json" -d '{"id":"3bde47e2-13a8-4ed8-a88e-1518c7e0dd00", "message":"netstat -an"}' http://localhost:8084/send
curl -X POST -H "Content-Type: application/json" -d '{"id":"6d5a76ff-812f-4d7b-adf3-9089cc1ffce6", "message":"netstat -an"}' http://localhost:8084/send
```

#### Broadcast Commands
```shell
curl -X POST -H "Content-Type: application/json" -d '{"message":"Control broadcast message via http api"}' http://localhost:8084/broadcast
```


### Notes

- There is a timeout set for 10 seconds on direct methods.

- Build protobuf (if needed)

```shell
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/syswatch.proto
```

- If you want to change the server name you can modify the server_alt_names in the data/x509/openssl.cnf file.


## Todo
- Websocket interface for accessing streaming data
- Proper connection handling, and reconnection logic