package config

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	convey.Convey("Given an environment with no environment variables set", t, func() {
		cfg, err := Get()

		convey.Convey("When the config values are retrieved", func() {
			convey.Convey("Then there should be no error returned", func() {
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("Then the values should be set to the expected defaults", func() {
				convey.So(cfg.BindAddr, convey.ShouldEqual, "localhost:24700")
				convey.So(cfg.APIURL, convey.ShouldResemble, "http://localhost:24700")
				convey.So(cfg.Brokers, convey.ShouldResemble, []string{"localhost:9092", "localhost:9093", "localhost:9094"})
				convey.So(cfg.KafkaVersion, convey.ShouldEqual, "1.0.2")
				convey.So(cfg.KafkaSecProtocol, convey.ShouldEqual, "")
				convey.So(cfg.KafkaMaxBytes, convey.ShouldEqual, 2000000)
				convey.So(cfg.ImageUploadedTopic, convey.ShouldEqual, "image-uploaded")
				convey.So(cfg.StaticFilePublishedTopic, convey.ShouldEqual, "static-file-published")
				convey.So(cfg.GracefulShutdownTimeout, convey.ShouldEqual, 5*time.Second)
				convey.So(cfg.HealthCheckInterval, convey.ShouldEqual, 30*time.Second)
				convey.So(cfg.HealthCheckCriticalTimeout, convey.ShouldEqual, 90*time.Second)
				convey.So(cfg.MongoConfig.ClusterEndpoint, convey.ShouldEqual, "localhost:27017")
				convey.So(cfg.MongoConfig.Database, convey.ShouldEqual, "images")
				convey.So(cfg.MongoConfig.Collections, convey.ShouldResemble, map[string]string{ImagesCollection: "images", ImagesLockCollection: "images_locks"})
				convey.So(cfg.MongoConfig.Username, convey.ShouldEqual, "")
				convey.So(cfg.MongoConfig.Password, convey.ShouldEqual, "")
				convey.So(cfg.MongoConfig.ReplicaSet, convey.ShouldEqual, "")
				convey.So(cfg.MongoConfig.IsStrongReadConcernEnabled, convey.ShouldEqual, false)
				convey.So(cfg.MongoConfig.IsWriteConcernMajorityEnabled, convey.ShouldEqual, true)
				convey.So(cfg.MongoConfig.QueryTimeout, convey.ShouldEqual, 15*time.Second)
				convey.So(cfg.MongoConfig.ConnectTimeout, convey.ShouldEqual, 5*time.Second)
				convey.So(cfg.MongoConfig.IsSSL, convey.ShouldEqual, false)
				convey.So(cfg.MongoConfig.VerifyCert, convey.ShouldEqual, false)
				convey.So(cfg.IsPublishing, convey.ShouldBeTrue)
				convey.So(cfg.ZebedeeURL, convey.ShouldEqual, "http://localhost:8082")
				convey.So(cfg.DownloadServiceURL, convey.ShouldEqual, "http://localhost:23600")
			})
			convey.Convey("Then a second call to config should return the same config", func() {
				newCfg, newErr := Get()
				convey.So(newErr, convey.ShouldBeNil)
				convey.So(newCfg, convey.ShouldResemble, cfg)
			})
		})
	})
}
