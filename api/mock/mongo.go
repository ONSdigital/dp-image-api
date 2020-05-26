// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"context"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/models"
	"sync"
)

var (
	lockMongoServerMockChecker     sync.RWMutex
	lockMongoServerMockClose       sync.RWMutex
	lockMongoServerMockGetImage    sync.RWMutex
	lockMongoServerMockGetImages   sync.RWMutex
	lockMongoServerMockUpdateImage sync.RWMutex
	lockMongoServerMockUpsertImage sync.RWMutex
)

// Ensure, that MongoServerMock does implement api.MongoServer.
// If this is not the case, regenerate this file with moq.
var _ api.MongoServer = &MongoServerMock{}

// MongoServerMock is a mock implementation of api.MongoServer.
//
//     func TestSomethingThatUsesMongoServer(t *testing.T) {
//
//         // make and configure a mocked api.MongoServer
//         mockedMongoServer := &MongoServerMock{
//             CheckerFunc: func(ctx context.Context, state *healthcheck.CheckState) error {
// 	               panic("mock out the Checker method")
//             },
//             CloseFunc: func(ctx context.Context) error {
// 	               panic("mock out the Close method")
//             },
//             GetImageFunc: func(ctx context.Context, id string) (*models.Image, error) {
// 	               panic("mock out the GetImage method")
//             },
//             GetImagesFunc: func(ctx context.Context, collectionID string) ([]models.Image, error) {
// 	               panic("mock out the GetImages method")
//             },
//             UpdateImageFunc: func(ctx context.Context, id string, image *models.Image) error {
// 	               panic("mock out the UpdateImage method")
//             },
//             UpsertImageFunc: func(ctx context.Context, id string, image *models.Image) error {
// 	               panic("mock out the UpsertImage method")
//             },
//         }
//
//         // use mockedMongoServer in code that requires api.MongoServer
//         // and then make assertions.
//
//     }
type MongoServerMock struct {
	// CheckerFunc mocks the Checker method.
	CheckerFunc func(ctx context.Context, state *healthcheck.CheckState) error

	// CloseFunc mocks the Close method.
	CloseFunc func(ctx context.Context) error

	// GetImageFunc mocks the GetImage method.
	GetImageFunc func(ctx context.Context, id string) (*models.Image, error)

	// GetImagesFunc mocks the GetImages method.
	GetImagesFunc func(ctx context.Context, collectionID string) ([]models.Image, error)

	// UpdateImageFunc mocks the UpdateImage method.
	UpdateImageFunc func(ctx context.Context, id string, image *models.Image) error

	// UpsertImageFunc mocks the UpsertImage method.
	UpsertImageFunc func(ctx context.Context, id string, image *models.Image) error

	// calls tracks calls to the methods.
	calls struct {
		// Checker holds details about calls to the Checker method.
		Checker []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// State is the state argument value.
			State *healthcheck.CheckState
		}
		// Close holds details about calls to the Close method.
		Close []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
		// GetImage holds details about calls to the GetImage method.
		GetImage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
		}
		// GetImages holds details about calls to the GetImages method.
		GetImages []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// CollectionID is the collectionID argument value.
			CollectionID string
		}
		// UpdateImage holds details about calls to the UpdateImage method.
		UpdateImage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Image is the image argument value.
			Image *models.Image
		}
		// UpsertImage holds details about calls to the UpsertImage method.
		UpsertImage []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// Image is the image argument value.
			Image *models.Image
		}
	}
}

// Checker calls CheckerFunc.
func (mock *MongoServerMock) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	if mock.CheckerFunc == nil {
		panic("MongoServerMock.CheckerFunc: method is nil but MongoServer.Checker was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		State *healthcheck.CheckState
	}{
		Ctx:   ctx,
		State: state,
	}
	lockMongoServerMockChecker.Lock()
	mock.calls.Checker = append(mock.calls.Checker, callInfo)
	lockMongoServerMockChecker.Unlock()
	return mock.CheckerFunc(ctx, state)
}

// CheckerCalls gets all the calls that were made to Checker.
// Check the length with:
//     len(mockedMongoServer.CheckerCalls())
func (mock *MongoServerMock) CheckerCalls() []struct {
	Ctx   context.Context
	State *healthcheck.CheckState
} {
	var calls []struct {
		Ctx   context.Context
		State *healthcheck.CheckState
	}
	lockMongoServerMockChecker.RLock()
	calls = mock.calls.Checker
	lockMongoServerMockChecker.RUnlock()
	return calls
}

// Close calls CloseFunc.
func (mock *MongoServerMock) Close(ctx context.Context) error {
	if mock.CloseFunc == nil {
		panic("MongoServerMock.CloseFunc: method is nil but MongoServer.Close was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	lockMongoServerMockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	lockMongoServerMockClose.Unlock()
	return mock.CloseFunc(ctx)
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//     len(mockedMongoServer.CloseCalls())
func (mock *MongoServerMock) CloseCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	lockMongoServerMockClose.RLock()
	calls = mock.calls.Close
	lockMongoServerMockClose.RUnlock()
	return calls
}

// GetImage calls GetImageFunc.
func (mock *MongoServerMock) GetImage(ctx context.Context, id string) (*models.Image, error) {
	if mock.GetImageFunc == nil {
		panic("MongoServerMock.GetImageFunc: method is nil but MongoServer.GetImage was just called")
	}
	callInfo := struct {
		Ctx context.Context
		ID  string
	}{
		Ctx: ctx,
		ID:  id,
	}
	lockMongoServerMockGetImage.Lock()
	mock.calls.GetImage = append(mock.calls.GetImage, callInfo)
	lockMongoServerMockGetImage.Unlock()
	return mock.GetImageFunc(ctx, id)
}

// GetImageCalls gets all the calls that were made to GetImage.
// Check the length with:
//     len(mockedMongoServer.GetImageCalls())
func (mock *MongoServerMock) GetImageCalls() []struct {
	Ctx context.Context
	ID  string
} {
	var calls []struct {
		Ctx context.Context
		ID  string
	}
	lockMongoServerMockGetImage.RLock()
	calls = mock.calls.GetImage
	lockMongoServerMockGetImage.RUnlock()
	return calls
}

// GetImages calls GetImagesFunc.
func (mock *MongoServerMock) GetImages(ctx context.Context, collectionID string) ([]models.Image, error) {
	if mock.GetImagesFunc == nil {
		panic("MongoServerMock.GetImagesFunc: method is nil but MongoServer.GetImages was just called")
	}
	callInfo := struct {
		Ctx          context.Context
		CollectionID string
	}{
		Ctx:          ctx,
		CollectionID: collectionID,
	}
	lockMongoServerMockGetImages.Lock()
	mock.calls.GetImages = append(mock.calls.GetImages, callInfo)
	lockMongoServerMockGetImages.Unlock()
	return mock.GetImagesFunc(ctx, collectionID)
}

// GetImagesCalls gets all the calls that were made to GetImages.
// Check the length with:
//     len(mockedMongoServer.GetImagesCalls())
func (mock *MongoServerMock) GetImagesCalls() []struct {
	Ctx          context.Context
	CollectionID string
} {
	var calls []struct {
		Ctx          context.Context
		CollectionID string
	}
	lockMongoServerMockGetImages.RLock()
	calls = mock.calls.GetImages
	lockMongoServerMockGetImages.RUnlock()
	return calls
}

// UpdateImage calls UpdateImageFunc.
func (mock *MongoServerMock) UpdateImage(ctx context.Context, id string, image *models.Image) error {
	if mock.UpdateImageFunc == nil {
		panic("MongoServerMock.UpdateImageFunc: method is nil but MongoServer.UpdateImage was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		ID    string
		Image *models.Image
	}{
		Ctx:   ctx,
		ID:    id,
		Image: image,
	}
	lockMongoServerMockUpdateImage.Lock()
	mock.calls.UpdateImage = append(mock.calls.UpdateImage, callInfo)
	lockMongoServerMockUpdateImage.Unlock()
	return mock.UpdateImageFunc(ctx, id, image)
}

// UpdateImageCalls gets all the calls that were made to UpdateImage.
// Check the length with:
//     len(mockedMongoServer.UpdateImageCalls())
func (mock *MongoServerMock) UpdateImageCalls() []struct {
	Ctx   context.Context
	ID    string
	Image *models.Image
} {
	var calls []struct {
		Ctx   context.Context
		ID    string
		Image *models.Image
	}
	lockMongoServerMockUpdateImage.RLock()
	calls = mock.calls.UpdateImage
	lockMongoServerMockUpdateImage.RUnlock()
	return calls
}

// UpsertImage calls UpsertImageFunc.
func (mock *MongoServerMock) UpsertImage(ctx context.Context, id string, image *models.Image) error {
	if mock.UpsertImageFunc == nil {
		panic("MongoServerMock.UpsertImageFunc: method is nil but MongoServer.UpsertImage was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		ID    string
		Image *models.Image
	}{
		Ctx:   ctx,
		ID:    id,
		Image: image,
	}
	lockMongoServerMockUpsertImage.Lock()
	mock.calls.UpsertImage = append(mock.calls.UpsertImage, callInfo)
	lockMongoServerMockUpsertImage.Unlock()
	return mock.UpsertImageFunc(ctx, id, image)
}

// UpsertImageCalls gets all the calls that were made to UpsertImage.
// Check the length with:
//     len(mockedMongoServer.UpsertImageCalls())
func (mock *MongoServerMock) UpsertImageCalls() []struct {
	Ctx   context.Context
	ID    string
	Image *models.Image
} {
	var calls []struct {
		Ctx   context.Context
		ID    string
		Image *models.Image
	}
	lockMongoServerMockUpsertImage.RLock()
	calls = mock.calls.UpsertImage
	lockMongoServerMockUpsertImage.RUnlock()
	return calls
}
