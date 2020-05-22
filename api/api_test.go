package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"

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
		api := api.Setup(ctx, r, &mock.MongoServerMock{})

		Convey("When created the following routes should have been added", func() {
			So(hasRoute(api.Router, "/images", http.MethodPost), ShouldBeTrue)
			So(hasRoute(api.Router, "/images", http.MethodGet), ShouldBeTrue)
			So(hasRoute(api.Router, "/images/{id}", http.MethodGet), ShouldBeTrue)
			So(hasRoute(api.Router, "/images/{id}", http.MethodPut), ShouldBeTrue)
			So(hasRoute(api.Router, "/images/{id}/publish", http.MethodPost), ShouldBeTrue)
		})
	})
}

func TestClose(t *testing.T) {
	Convey("Given an API instance", t, func() {
		r := mux.NewRouter()
		ctx := context.Background()
		a := api.Setup(ctx, r, &mock.MongoServerMock{})

		Convey("When the api is closed any dependencies are closed also", func() {
			err := a.Close(ctx)
			So(err, ShouldBeNil)
			// Check that dependencies are closed here
		})
	})
}

// GetAPIWithMocks also used in other tests
func GetAPIWithMocks(mongoDbMock api.MongoServer, cfg *config.Config) *api.API {
	mu.Lock()
	defer mu.Unlock()
	return api.Setup(testContext, mux.NewRouter(), mongoDbMock)
}

func hasRoute(r *mux.Router, path, method string) bool {
	req := httptest.NewRequest(method, path, nil)
	match := &mux.RouteMatch{}
	return r.Match(req, match)
}
