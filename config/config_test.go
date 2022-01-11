package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("Given an environment with no environment variables set", t, func() {
		cfg, err := Get()

		Convey("When the config values are retrieved", func() {

			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, "localhost:24700")
				So(cfg.ApiURL, ShouldResemble, "http://localhost:24700")
				So(cfg.Brokers, ShouldResemble, []string{"localhost:9092", "localhost:9093", "localhost:9094"})
				So(cfg.KafkaVersion, ShouldEqual, "1.0.2")
				So(cfg.KafkaSecProtocol, ShouldEqual, "")
				So(cfg.KafkaMaxBytes, ShouldEqual, 2000000)
				So(cfg.ImageUploadedTopic, ShouldEqual, "image-uploaded")
				So(cfg.StaticFilePublishedTopic, ShouldEqual, "static-file-published")
				So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(cfg.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
				So(cfg.MongoConfig.ClusterEndpoint, ShouldEqual, "localhost:27017")
				So(cfg.MongoConfig.Database, ShouldEqual, "images")
				So(cfg.MongoConfig.Collections, ShouldResemble, map[string]string{ImagesCollection: "images", ImagesLockCollection: "images_locks"})
				So(cfg.MongoConfig.Username, ShouldEqual, "")
				So(cfg.MongoConfig.Password, ShouldEqual, "")
				So(cfg.MongoConfig.ReplicaSet, ShouldEqual, "")
				So(cfg.MongoConfig.IsStrongReadConcernEnabled, ShouldEqual, false)
				So(cfg.MongoConfig.IsWriteConcernMajorityEnabled, ShouldEqual, true)
				So(cfg.MongoConfig.QueryTimeout, ShouldEqual, 15*time.Second)
				So(cfg.MongoConfig.ConnectTimeout, ShouldEqual, 5*time.Second)
				So(cfg.MongoConfig.IsSSL, ShouldEqual, false)
				So(cfg.MongoConfig.VerifyCert, ShouldEqual, false)
				So(cfg.IsPublishing, ShouldBeTrue)
				So(cfg.ZebedeeURL, ShouldEqual, "http://localhost:8082")
				So(cfg.DownloadServiceURL, ShouldEqual, "http://localhost:23600")

			})

			Convey("Then a second call to config should return the same config", func() {
				newCfg, newErr := Get()
				So(newErr, ShouldBeNil)
				So(newCfg, ShouldResemble, cfg)
			})
		})
	})
}
