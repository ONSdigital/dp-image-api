dp-image-api
================
Digital Publishing Image API

### Getting started

* Run `make debug`

### Dependencies

* No further dependencies other than those defined in `go.mod`

### Configuration

| Environment variable         | Default               | Description
| ---------------------------- | --------------------- | -----------
| BIND_ADDR                    | :24700                | The host and port to bind to
| KAFKA_ADDR                   | localhost:9092        | The list of kafka broker hosts (publishing mode only)
| KAFKA_MAX_BYTES              | 2000000               | Maximum number of bytes in a kafka message (publishing mode only)
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

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2021, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.

