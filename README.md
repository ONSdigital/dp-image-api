# dp-image-api

Digital Publishing Image API

## Getting started

* Run `make debug`

### Dependencies

* No further dependencies other than those defined in `go.mod`

### Configuration

| Environment variable         | Default               | Description
| ---------------------------- | --------------------- | -----------
| BIND_ADDR                    | :24700                | The host and port to bind to
| KAFKA_ADDR                   | localhost:9092        | The list of kafka broker hosts (publishing mode only)
| KAFKA_VERSION                | `1.0.2`                                | The version of (TLS-ready) Kafka
| KAFKA_MAX_BYTES              | 2000000               | Maximum number of bytes in a kafka message (publishing mode only)
| KAFKA_SEC_PROTO              | _unset_               | if set to `TLS`, kafka connections will use TLS [1]
| KAFKA_SEC_CLIENT_KEY         | _unset_               | PEM for the client key [1]
| KAFKA_SEC_CLIENT_CERT        | _unset_               | PEM for the client certificate [1]
| KAFKA_SEC_CA_CERTS           | _unset_               | CA cert chain for the server cert [1]
| KAFKA_SEC_SKIP_VERIFY        | false                 | ignores server certificate issues if `true` [1]
| IMAGE_UPLOADED_TOPIC         | image-uploaded        | The kafka topic that will be produced by this service for image uploading events (publishing mode only)
| STATIC_FILE_PUBLISHED_TOPIC  | static-file-published | The kafka topic that will be produced by this service for image publishing events (publishing mode only)
| GRACEFUL_SHUTDOWN_TIMEOUT    | 5s                    | The graceful shutdown timeout in seconds (`time.Duration` format)
| HEALTHCHECK_INTERVAL         | 30s                   | Time between self-healthchecks (`time.Duration` format)
| HEALTHCHECK_CRITICAL_TIMEOUT | 90s                   | Time to wait until an unhealthy dependent propagates its state to make this app unhealthy (`time.Duration` format)
| IS_PUBLISHING                | true                  | Determines if the instance is publishing or not
| ZEBEDEE_URL                  | http://localhost:8082 | The URL of zebedee (publishing mode only)
| MONGODB_BIND_ADDR            | localhost:27017       | The MongoDB bind address
| MONGODB_COLLECTION           | images                | The MongoDB images database
| MONGODB_DATABASE             | images                | MongoDB collection
| MONGODB_USERNAME             | test                  | MongoDB Username
| MONGODB_PASSWORD             | test                  | MongoDB Password
| MONGODB_IS_SSL               | false                 | is SSL enabled for mongo server

**Notes:**

1. For more info, see the [kafka TLS examples documentation](https://github.com/ONSdigital//tree/main/examples#tls)

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2021, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.

