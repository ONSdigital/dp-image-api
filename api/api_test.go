package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ONSdigital/dp-image-api/url"

	"github.com/gorilla/mux"

	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/config"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/dp-kafka/v3/kafkatest"
	"github.com/smartystreets/goconvey/convey"
)

var (
	mu          sync.Mutex
	testContext = context.Background()
)

func TestSetup(t *testing.T) {
	convey.Convey("Given an API instance", t, func() {
		r := mux.NewRouter()
		ctx := context.Background()

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(_ dpauth.Permissions, _ http.HandlerFunc) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
		}
		urlBuilder := url.NewBuilder("")

		convey.Convey("When created in Publishing mode", func() {
			cfg := &config.Config{IsPublishing: true}
			uploadedKafkaProducer := &kafkatest.IProducerMock{
				ChannelsFunc: func() *kafka.ProducerChannels {
					return &kafka.ProducerChannels{}
				},
			}
			publishedKafkaProducer := &kafkatest.IProducerMock{
				ChannelsFunc: func() *kafka.ProducerChannels {
					return &kafka.ProducerChannels{}
				},
			}
			api := api.Setup(ctx, cfg, r, authHandlerMock, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer, urlBuilder)

			convey.Convey("Then the following routes should have been added", func() {
				convey.So(hasRoute(api.Router, "/images", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images", http.MethodPost), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}", http.MethodPut), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads", http.MethodPost), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads/{variant}", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads/{variant}", http.MethodPut), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPost), convey.ShouldBeTrue)
			})

			convey.Convey("And auth handler is called once per route with the expected permissions", func() {
				convey.So(authHandlerMock.RequireCalls(), convey.ShouldHaveLength, 9)
				convey.So(authHandlerMock.RequireCalls()[0].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images
				convey.So(authHandlerMock.RequireCalls()[1].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: true, Read: false, Update: false, Delete: false}) // permissions for POST /images
				convey.So(authHandlerMock.RequireCalls()[2].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images/{id}
				convey.So(authHandlerMock.RequireCalls()[3].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for PUT /images/{id}
				convey.So(authHandlerMock.RequireCalls()[4].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images/{id}/downloads
				convey.So(authHandlerMock.RequireCalls()[5].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/downloads
				convey.So(authHandlerMock.RequireCalls()[6].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images/{id}/downloads/{variant}
				convey.So(authHandlerMock.RequireCalls()[7].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for PUT /images/{id}/downloads/{variant}
				convey.So(authHandlerMock.RequireCalls()[8].Required, convey.ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/publish
			})
		})

		convey.Convey("When created in Web mode", func() {
			cfg := &config.Config{IsPublishing: false}
			uploadedKafkaProducer := &kafkatest.IProducerMock{}
			publishedKafkaProducer := &kafkatest.IProducerMock{}
			api := api.Setup(ctx, cfg, r, authHandlerMock, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer, urlBuilder)

			convey.Convey("Then only the get routes should have been added", func() {
				convey.So(hasRoute(api.Router, "/images", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images", http.MethodPost), convey.ShouldBeFalse)
				convey.So(hasRoute(api.Router, "/images/{id}", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}", http.MethodPut), convey.ShouldBeFalse)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads", http.MethodPost), convey.ShouldBeFalse)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads/{variant}", http.MethodGet), convey.ShouldBeTrue)
				convey.So(hasRoute(api.Router, "/images/{id}/downloads/{variant}", http.MethodPut), convey.ShouldBeFalse)
				convey.So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPut), convey.ShouldBeFalse)
			})

			convey.Convey("And no auth permissions are required", func() {
				convey.So(authHandlerMock.RequireCalls(), convey.ShouldHaveLength, 0)
			})
		})
	})
}

func TestClose(t *testing.T) {
	convey.Convey("Given an API instance", t, func() {
		r := mux.NewRouter()
		ctx := context.Background()
		uploadedKafkaProducer := &kafkatest.IProducerMock{}
		publishedKafkaProducer := &kafkatest.IProducerMock{}
		urlBuilder := url.NewBuilder("")
		a := api.Setup(ctx, &config.Config{}, r, &mock.AuthHandlerMock{}, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer, urlBuilder)

		convey.Convey("When the api is closed any dependencies are closed also", func() {
			err := a.Close(ctx)
			convey.So(err, convey.ShouldBeNil)
			// Check that dependencies are closed here
		})
	})
}

// GetAPIWithMocks also used in other tests
func GetAPIWithMocks(cfg *config.Config, mongoDBMock *mock.MongoServerMock, authHandlerMock *mock.AuthHandlerMock, uploadedKafkaProducerMock, publishedKafkaProducerMock kafka.IProducer) *api.API {
	mu.Lock()
	defer mu.Unlock()
	urlBuilder := url.NewBuilder("http://example.com")
	return api.Setup(testContext, cfg, mux.NewRouter(), authHandlerMock, mongoDBMock, uploadedKafkaProducerMock, publishedKafkaProducerMock, urlBuilder)
}

func hasRoute(r *mux.Router, path, method string) bool {
	req := httptest.NewRequest(method, path, http.NoBody)
	match := &mux.RouteMatch{}
	return r.Match(req, match)
}
