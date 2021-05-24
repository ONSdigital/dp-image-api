module github.com/ONSdigital/dp-image-api

go 1.15

require (
	github.com/ONSdigital/dp-api-clients-go v1.28.0
	github.com/ONSdigital/dp-authorisation v0.1.0
	github.com/ONSdigital/dp-healthcheck v1.0.5
	github.com/ONSdigital/dp-kafka v1.1.6
	github.com/ONSdigital/dp-mongodb v1.5.0
	github.com/ONSdigital/dp-net v1.0.9
	github.com/ONSdigital/go-ns v0.0.0-20200205115900-a11716f93bad
	github.com/ONSdigital/log.go v1.0.1
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gorilla/mux v1.8.0
	github.com/justinas/alice v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/smartystreets/goconvey v1.6.4
	gopkg.in/avro.v0 v0.0.0-20171217001914-a730b5802183 // indirect
)

replace github.com/ONSdigital/dp-mongodb v1.5.0 => github.com/ONSdigital/dp-mongodb v1.5.1-0.20210524082442-30ac58e9cbec
