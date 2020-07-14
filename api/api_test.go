package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"

	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/config"
	kafka "github.com/ONSdigital/dp-kafka"
	"github.com/ONSdigital/dp-kafka/kafkatest"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	mu          sync.Mutex
	testContext = context.Background()
)

func TestSetup(t *testing.T) {
	Convey("Given an API instance", t, func() {
		r := mux.NewRouter()
		ctx := context.Background()

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
		}

		Convey("When created in Publishing mode", func() {
			cfg := &config.Config{IsPublishing: true}
			uploadedKafkaProducer := kafkatest.NewMessageProducer(true)
			publishedKafkaProducer := kafkatest.NewMessageProducer(true)
			api := api.Setup(ctx, cfg, r, authHandlerMock, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer)

			Convey("Then the following routes should have been added", func() {
				So(hasRoute(api.Router, "/images", http.MethodPost), ShouldBeTrue)
				So(hasRoute(api.Router, "/images", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodPut), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}/upload", http.MethodPost), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPost), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}/downloads/{variant}/import", http.MethodPost), ShouldBeTrue)
			})

			Convey("And auth handler is called once per route with the expected permissions", func() {
				So(len(authHandlerMock.RequireCalls()), ShouldEqual, 7)
				So(authHandlerMock.RequireCalls()[0].Required, ShouldResemble, dpauth.Permissions{
					Create: true, Read: false, Update: false, Delete: false}) // permissions for POST /images
				So(authHandlerMock.RequireCalls()[1].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images
				So(authHandlerMock.RequireCalls()[2].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images/{id}
				So(authHandlerMock.RequireCalls()[3].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for PUT /images/{id}
				So(authHandlerMock.RequireCalls()[4].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/upload
				So(authHandlerMock.RequireCalls()[5].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/publish
				So(authHandlerMock.RequireCalls()[5].Required, ShouldResemble, dpauth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/downloads/{variant}/import
			})
		})

		Convey("When created in Web mode", func() {
			cfg := &config.Config{IsPublishing: false}
			uploadedKafkaProducer := kafkatest.NewMessageProducer(true)
			publishedKafkaProducer := kafkatest.NewMessageProducer(true)
			api := api.Setup(ctx, cfg, r, authHandlerMock, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer)

			Convey("Then only the get routes should have been added", func() {
				So(hasRoute(api.Router, "/images", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images", http.MethodPost), ShouldBeFalse)
				So(hasRoute(api.Router, "/images/{id}", http.MethodPut), ShouldBeFalse)
				So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPut), ShouldBeFalse)
			})

			Convey("And no auth permissions are required", func() {
				So(len(authHandlerMock.RequireCalls()), ShouldEqual, 0)
			})
		})
	})
}

func TestClose(t *testing.T) {
	Convey("Given an API instance", t, func() {
		r := mux.NewRouter()
		ctx := context.Background()
		uploadedKafkaProducer := kafkatest.NewMessageProducer(true)
		publishedKafkaProducer := kafkatest.NewMessageProducer(true)
		a := api.Setup(ctx, &config.Config{}, r, &mock.AuthHandlerMock{}, &mock.MongoServerMock{}, uploadedKafkaProducer, publishedKafkaProducer)

		Convey("When the api is closed any dependencies are closed also", func() {
			err := a.Close(ctx)
			So(err, ShouldBeNil)
			// Check that dependencies are closed here
		})
	})
}

// GetAPIWithMocks also used in other tests
func GetAPIWithMocks(cfg *config.Config, mongoDbMock *mock.MongoServerMock, authHandlerMock *mock.AuthHandlerMock, uploadedKafkaProducerMock, publishedKafkaProducerMock kafka.IProducer) *api.API {
	mu.Lock()
	defer mu.Unlock()
	return api.Setup(testContext, cfg, mux.NewRouter(), authHandlerMock, mongoDbMock, uploadedKafkaProducerMock, publishedKafkaProducerMock)
}

func hasRoute(r *mux.Router, path, method string) bool {
	req := httptest.NewRequest(method, path, nil)
	match := &mux.RouteMatch{}
	return r.Match(req, match)
}
