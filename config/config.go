package config

import (
	"time"

	mongodriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"

	"github.com/kelseyhightower/envconfig"
)

type MongoConfig = mongodriver.MongoConnectionConfig

// Config represents service configuration for dp-image-api
type Config struct {
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	ApiURL                     string        `envconfig:"IMAGE_API_URL"`
	Brokers                    []string      `envconfig:"KAFKA_ADDR"`
	KafkaMaxBytes              int           `envconfig:"KAFKA_MAX_BYTES"`
	KafkaVersion               string        `envconfig:"KAFKA_VERSION"`
	KafkaSecProtocol           string        `envconfig:"KAFKA_SEC_PROTO"`
	KafkaSecCACerts            string        `envconfig:"KAFKA_SEC_CA_CERTS"`
	KafkaSecClientCert         string        `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	KafkaSecClientKey          string        `envconfig:"KAFKA_SEC_CLIENT_KEY"             json:"-"`
	KafkaSecSkipVerify         bool          `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	ImageUploadedTopic         string        `envconfig:"IMAGE_UPLOADED_TOPIC"`
	StaticFilePublishedTopic   string        `envconfig:"STATIC_FILE_PUBLISHED_TOPIC"`
	GracefulShutdownTimeout    time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	IsPublishing               bool          `envconfig:"IS_PUBLISHING"`
	ZebedeeURL                 string        `envconfig:"ZEBEDEE_URL"`
	DownloadServiceURL         string        `envconfig:"DOWNLOAD_SERVICE_URL"`
	MongoConfig
}

var cfg *Config

// Get returns the default config with any modifications through environment
// variables
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg := &Config{
		BindAddr:                   "localhost:24700",
		ApiURL:                     "http://localhost:24700",
		Brokers:                    []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		KafkaVersion:               "1.0.2",
		KafkaMaxBytes:              2000000,
		ImageUploadedTopic:         "image-uploaded",
		StaticFilePublishedTopic:   "static-file-published",
		GracefulShutdownTimeout:    5 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		ZebedeeURL:                 "http://localhost:8082",
		IsPublishing:               true,
		DownloadServiceURL:         "http://localhost:23600",
		MongoConfig: MongoConfig{
			ClusterEndpoint:               "localhost:27017",
			Username:                      "",
			Password:                      "",
			Database:                      "images",
			Collection:                    "images",
			ReplicaSet:                    "",
			IsStrongReadConcernEnabled:    false,
			IsWriteConcernMajorityEnabled: true,
			ConnectTimeoutInSeconds:       5 * time.Second,
			QueryTimeoutInSeconds:         15 * time.Second,
			TLSConnectionConfig: mongodriver.TLSConnectionConfig{
				IsSSL: false,
			},
		},
	}

	return cfg, envconfig.Process("", cfg)
}
