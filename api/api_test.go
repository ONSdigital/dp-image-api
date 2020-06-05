package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"

	"github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/config"
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
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return func(http.ResponseWriter, *http.Request) {}
			},
		}

		Convey("When created in Publishing mode", func() {
			cfg := &config.Config{IsPublishing: true}
			api := api.Setup(ctx, cfg, r, &mock.MongoServerMock{}, authHandlerMock)

			Convey("Then the following routes should have been added", func() {
				So(hasRoute(api.Router, "/images", http.MethodPost), ShouldBeTrue)
				So(hasRoute(api.Router, "/images", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodPut), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPost), ShouldBeTrue)
			})

			Convey("And auth handler is called once per route with the expected permissions", func() {
				So(len(authHandlerMock.RequireCalls()), ShouldEqual, 5)
				So(authHandlerMock.RequireCalls()[0].Required, ShouldResemble, auth.Permissions{
					Create: true, Read: false, Update: false, Delete: false}) // permissions for POST /images
				So(authHandlerMock.RequireCalls()[1].Required, ShouldResemble, auth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images
				So(authHandlerMock.RequireCalls()[2].Required, ShouldResemble, auth.Permissions{
					Create: false, Read: true, Update: false, Delete: false}) // permissions for GET /images/{id}
				So(authHandlerMock.RequireCalls()[3].Required, ShouldResemble, auth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for PUT /images/{id}
				So(authHandlerMock.RequireCalls()[4].Required, ShouldResemble, auth.Permissions{
					Create: false, Read: false, Update: true, Delete: false}) // permissions for POST /images/{id}/publish
			})
		})

		Convey("When created in Web mode", func() {
			cfg := &config.Config{IsPublishing: false}
			api := api.Setup(ctx, cfg, r, &mock.MongoServerMock{}, authHandlerMock)

			Convey("Then only the get routes should have been added", func() {
				So(hasRoute(api.Router, "/images", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images/{id}", http.MethodGet), ShouldBeTrue)
				So(hasRoute(api.Router, "/images", http.MethodPost), ShouldBeFalse)
				So(hasRoute(api.Router, "/images/{id}", http.MethodPut), ShouldBeFalse)
				So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPost), ShouldBeFalse)
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
		a := api.Setup(ctx, &config.Config{}, r, &mock.MongoServerMock{}, &mock.AuthHandlerMock{})

		Convey("When the api is closed any dependencies are closed also", func() {
			err := a.Close(ctx)
			So(err, ShouldBeNil)
			// Check that dependencies are closed here
		})
	})
}

// GetAPIWithMocks also used in other tests
func GetAPIWithMocks(cfg *config.Config, mongoDbMock api.MongoServer, authHandlerMock *mock.AuthHandlerMock) *api.API {
	mu.Lock()
	defer mu.Unlock()
	return api.Setup(testContext, &config.Config{}, mux.NewRouter(), mongoDbMock, authHandlerMock)
}

func hasRoute(r *mux.Router, path, method string) bool {
	req := httptest.NewRequest(method, path, nil)
	match := &mux.RouteMatch{}
	return r.Match(req, match)
}
