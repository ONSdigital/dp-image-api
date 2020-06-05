package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/dp-net/handlers"
	dphttp "github.com/ONSdigital/dp-net/http"

	. "github.com/smartystreets/goconvey/convey"
)

// Contants for testing
const (
	testUserAuthToken = "UserToken"
	testImageID1      = "imageImageID1"
	testImageID2      = "imageImageID2"
	testCollectionID1 = "1234"
	testCollectionID2 = "4321"
)

var errMongoDB = errors.New("MongoDB generic error")

// Variants of new Image Payload without any extra field.
var (
	newImagePayloadFmt = `{
		"collection_id": "%s",
		"filename": "%s",
		"license": {
			"title": "Open Government Licence v3.0",
			"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
		},
		"type": "chart"
	}`
	newImagePayload        = fmt.Sprintf(newImagePayloadFmt, testCollectionID1, "some-image-name")
	newImagePayloadTooLong = fmt.Sprintf(newImagePayloadFmt, testCollectionID1, "Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch")
)

// New Image Payload with state
var (
	newImageWithStatePayloadFmt = `{
		"collection_id": "%s",
		"filename": "some-image-name",
		"state": "%s",
		"license": {
			"title": "Open Government Licence v3.0",
			"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
		},
		"type": "chart"
	}`
	newImageInvalidStatePayload = fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, "invalidState")
	noIDColID1ImagePayload      = fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID1, models.StateCreated.String())
	noIDColID2ImagePayload      = fmt.Sprintf(newImageWithStatePayloadFmt, testCollectionID2, models.StateCreated.String())
	noIDNoColIDImagePayload     = fmt.Sprintf(newImageWithStatePayloadFmt, "", models.StateCreated.String())
)

// Full Image payload, containing all possible fields
var (
	fullImagePayloadFmt = `{
		"id": "%s",
		"collection_id": "%s",
		"filename": "some-image-name",
		"license": {
			"title": "Open Government Licence v3.0",
			"href": "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/"
		},
		"type": "chart",
		"state": "published",
		"upload": {
			"path": "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png"
		},
		"downloads": {
			"png": {
				"1920x1080": {
					"size": 1024,
					"href": "http://download.ons.gov.uk/images/042e216a-7822-4fa0-a3d6-e3f5248ffc35/image-name.png",
					"public": "my-public-bucket",
					"private": "my-private-bucket"
				}
			}
		}
	}`
	fullImagePayload          = fmt.Sprintf(fullImagePayloadFmt, testImageID2, testCollectionID1)
	fullImagePayloadCreatedID = fmt.Sprintf(fullImagePayloadFmt, testImageID1, testCollectionID1)
)

var createdImage = models.Image{
	ID:           testImageID1,
	CollectionID: testCollectionID1,
	Filename:     "some-image-name",
	License: &models.License{
		Title: "Open Government Licence v3.0",
		Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	},
	Type:  "chart",
	State: models.StateCreated.String(),
}

var publishedImage = models.Image{
	ID:           testImageID2,
	CollectionID: testCollectionID1,
	Filename:     "some-image-name",
	License: &models.License{
		Title: "Open Government Licence v3.0",
		Href:  "https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
	},
	Type: "chart",
	Upload: &models.Upload{
		Path: "images/025a789c-533f-4ecf-a83b-65412b96b2b7/image-name.png",
	},
	Downloads: map[string]map[string]models.Download{
		"png": {
			"1920x1080": models.Download{
				Size:    1024,
				Href:    "http://download.ons.gov.uk/images/042e216a-7822-4fa0-a3d6-e3f5248ffc35/image-name.png",
				Public:  "my-public-bucket",
				Private: "my-private-bucket",
			},
		},
	},
	State: models.StatePublished.String(),
}

var images = models.Images{
	Items:      []models.Image{createdImage, publishedImage},
	Count:      2,
	Limit:      2,
	TotalCount: 2,
	Offset:     0,
}

var emptyImages = models.Images{
	Items:      []models.Image{},
	Count:      0,
	Limit:      0,
	TotalCount: 0,
	Offset:     0,
}

func TestCreateImageHandler(t *testing.T) {

	api.NewID = func() string { return testImageID1 }

	Convey("Given an image API in publishing mode that can successfully store valid images in mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDbMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return nil },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}

		imageApi := GetAPIWithMocks(cfg, mongoDbMock, authHandlerMock)

		Convey("When a valid new image is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When a valid new image with extra fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(fullImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When a new image with an invalid state fields is posted", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImageInvalidStatePayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then a newly created image with the new id and provided details is returned with status code 201, ignoring the state and any field that is not supposed to be provided at creation time", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("Posting an image with a filename longer than the maximum allowed results in BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayloadTooLong))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("An empty request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("An invalid request body results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString("invalidJson"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})
	})

	Convey("Given an image API with mongoDB that fails to insert images", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		cfg.IsPublishing = true

		mongoDbMock := &mock.MongoServerMock{
			UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error { return errMongoDB },
		}

		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDbMock, authHandlerMock)

		Convey("When a new image is posted a 501 InternalServerError status code is returned", func() {
			r := httptest.NewRequest(http.MethodPost, "http://localhost:24700/images", bytes.NewBufferString(newImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestGetImageHandler(t *testing.T) {

	Convey("Given an image API in publishing mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = true
		So(err, ShouldBeNil)
		doTestGetImageHandler(cfg)
	})

	Convey("Given an image API in web mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = false
		So(err, ShouldBeNil)
		doTestGetImageHandler(cfg)
	})
}

func doTestGetImageHandler(cfg *config.Config) {

	Convey("And an image API with mongoDB returning 'created' and 'published' images", func() {

		mongoDbMock := &mock.MongoServerMock{
			GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
				switch id {
				case testImageID1:
					return &createdImage, nil
				case testImageID2:
					return &publishedImage, nil
				default:
					return nil, apierrors.ErrImageNotFound
				}
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDbMock, authHandlerMock)

		Convey("When an existing 'created' image is requested with the valid Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the expected image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, createdImage)
			})
		})

		Convey("When an existing 'published' image is requested without a Collection-Id context value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testImageID2), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the published image is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImage := models.Image{}
				err = json.Unmarshal(payload, &retImage)
				So(err, ShouldBeNil)
				So(retImage, ShouldResemble, publishedImage)
			})
		})

		Convey("Missing a valid Collection-Id when requesting a non-published image results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Requesting an inexistent image ID results in a NotFound response", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images/inexistent", nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotFound)
		})
	})
}

func TestGetImagesHandler(t *testing.T) {

	Convey("Given an image API in publishing mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = true
		So(err, ShouldBeNil)
		doTestGetImagesHandler(cfg)
	})

	Convey("Given an image API in web mode", t, func() {
		cfg, err := config.Get()
		cfg.IsPublishing = false
		So(err, ShouldBeNil)
		doTestGetImagesHandler(cfg)
	})
}

func doTestGetImagesHandler(cfg *config.Config) {

	Convey("And an image API with mongoDB returning the images for the testing Collection-Id", func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDbMock := &mock.MongoServerMock{
			GetImagesFunc: func(ctx context.Context, collectionID string) ([]models.Image, error) {
				if collectionID == testCollectionID1 {
					return []models.Image{createdImage, publishedImage}, nil
				}
				return []models.Image{}, nil
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDbMock, authHandlerMock)

		Convey("When existing images are requested with a valid Collection-Id context and query value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then the expected images are returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, images)
			})
		})

		Convey("When inexistent images are requested with a valid Collection-Id context and query value", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", "otherCollectionId"), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			Convey("Then an empty list of images is returned with status code 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
				payload, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				retImages := models.Images{}
				err = json.Unmarshal(payload, &retImages)
				So(err, ShouldBeNil)
				So(retImages, ShouldResemble, emptyImages)
			})
		})

		Convey("Missing a valid Collection-Id header when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Missing a valid Collection-Id query parmeter when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, "http://localhost:24700/images", nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Providing different values for Collection-Id header and query parameter when requesting images results in a BadRequest response", func() {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:24700/images?collection_id=%s", testCollectionID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "otherCollectionId"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

	})
}

func TestUpdateImageHandler(t *testing.T) {
	Convey("Given an image API with existing images", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		mongoDBMock := &mock.MongoServerMock{
			GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
				switch id {
				case testImageID1, "idUpdateImageErr":
					return &createdImage, nil
				case testImageID2:
					return &publishedImage, nil
				case "idGetImageErr":
					return nil, errors.New("internal mongoDB error")
				default:
					return nil, apierrors.ErrImageNotFound
				}
			},
			UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) error {
				switch id {
				case testImageID1, testImageID2:
					return nil
				case "idUpdateImageErr":
					return errors.New("internal mongoDB error")
				default:
					return apierrors.ErrImageNotFound
				}
			},
		}
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, mongoDBMock, authHandlerMock)

		Convey("Calling update image with a valid image results in 200 OK response with the expected image provided to mongoDB", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(noIDColID1ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusOK)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, testImageID1)
			So(*mongoDBMock.UpdateImageCalls()[0].Image, ShouldResemble, createdImage)
		})

		Convey("Calling update image with an image without collectionID results in 400 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(noIDNoColIDImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 0)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with an image that has a different id results in 400 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(fullImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 0)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with an image that has a filename that is too long results in 400 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(newImagePayloadTooLong))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 0)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with an inexistent image id results in 404 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", "inexistent"), bytes.NewBufferString(noIDColID1ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotFound)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, "inexistent")
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with a collectionID that does not match the header collectionID results in 400 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(noIDColID1ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), "differentCollectionID"))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 0)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with a collectionID that does not match the mongoDB collectionID results in 400 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(noIDColID2ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID2))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusBadRequest)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image with a not-allowed state transition results in 409 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID1), bytes.NewBufferString(fullImagePayloadCreatedID))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusConflict)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID1)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("Calling update image for an already published image results in 409 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", testImageID2), bytes.NewBufferString(fullImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusConflict)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, testImageID2)
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("An internal mongoDB error in getImage results in 500 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", "idGetImageErr"), bytes.NewBufferString(noIDColID1ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, "idGetImageErr")
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 0)
		})

		Convey("An internal mongoDB error in updateImage results in 500 response", func() {
			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("http://localhost:24700/images/%s", "idUpdateImageErr"), bytes.NewBufferString(noIDColID1ImagePayload))
			r = r.WithContext(context.WithValue(r.Context(), handlers.CollectionID.Context(), testCollectionID1))
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusInternalServerError)
			So(len(mongoDBMock.GetImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.GetImageCalls()[0].ID, ShouldEqual, "idUpdateImageErr")
			So(len(mongoDBMock.UpdateImageCalls()), ShouldEqual, 1)
			So(mongoDBMock.UpdateImageCalls()[0].ID, ShouldEqual, "idUpdateImageErr")
		})
	})

}

func TestPublishImageHandler(t *testing.T) {
	Convey("Given an image API with empty mongoDB", t, func() {
		cfg, err := config.Get()
		So(err, ShouldBeNil)
		authHandlerMock := &mock.AuthHandlerMock{
			RequireFunc: func(required auth.Permissions, handler http.HandlerFunc) http.HandlerFunc {
				return handler
			},
		}
		imageApi := GetAPIWithMocks(cfg, &mock.MongoServerMock{}, authHandlerMock)

		Convey("Calling publish image results in 501 NotImplemented", func() {
			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:24700/images/%s/publish", testImageID1), nil)
			r = r.WithContext(context.WithValue(r.Context(), dphttp.FlorenceIdentityKey, testUserAuthToken))
			w := httptest.NewRecorder()
			imageApi.Router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, http.StatusNotImplemented)
		})
	})
}
