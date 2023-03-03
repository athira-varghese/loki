package ibmcloud

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/IBM/ibm-cos-sdk-go/aws/request"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/IBM/ibm-cos-sdk-go/service/s3/s3iface"
	"github.com/grafana/dskit/backoff"

	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/pkg/storage/chunk/client"
	"github.com/grafana/loki/pkg/storage/chunk/client/hedging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	bucket   = "test"
	testData = map[string][]byte{
		"key-1": []byte("test data 1"),
		"key-2": []byte("test data 2"),
		"key-3": []byte("test data 3"),
	}
	errMissingBucket = errors.New("bucket not found")
	errMissingKey    = errors.New("key not found")
	errMissingObject = errors.New("Object data not found")
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type mockCosClient struct {
	s3iface.S3API
	data   map[string][]byte
	bucket string
}

func newMockCosClient() *mockCosClient {
	return &mockCosClient{
		data:   testData,
		bucket: bucket,
	}
}

func Test_DeleteObject(t *testing.T) {
	tests := []struct {
		key     string
		wantErr error
	}{
		{
			"key-1",
			nil,
		},
		{
			"key-4",
			errMissingObject,
		},
	}

	for _, tt := range tests {
		cosConfig := COSConfig{
			BucketNames:     bucket,
			Endpoint:        "test",
			Region:          "dummy",
			AccessKeyID:     "dummy",
			SecretAccessKey: flagext.SecretWithValue("dummy"),
			BackoffConfig: backoff.Config{
				MaxRetries: 1,
			},
		}
		cosClient, err := NewCOSObjectClient(cosConfig, hedging.Config{})
		require.NoError(t, err)
		cosClient.cos = newMockCosClient()
		err = cosClient.DeleteObject(context.Background(), tt.key)
		if tt.wantErr != nil {
			require.Equal(t, tt.wantErr.Error(), err.Error())
			continue
		}
		require.NoError(t, err)
	}
}

func Test_List(t *testing.T) {
	tests := []struct {
		prefix      string
		delimiter   string
		storage_obj []client.StorageObject
		wantErr     error
	}{
		{
			"key",
			"-",
			[]client.StorageObject{},
			nil,
		},
	}

	for _, tt := range tests {
		cosConfig := COSConfig{
			BucketNames:     bucket,
			Endpoint:        "test",
			Region:          "dummy",
			AccessKeyID:     "dummy",
			SecretAccessKey: flagext.SecretWithValue("dummy"),
			BackoffConfig: backoff.Config{
				MaxRetries: 1,
			},
		}
		cosClient, err := NewCOSObjectClient(cosConfig, hedging.Config{})
		require.NoError(t, err)
		cosClient.cos = newMockCosClient()

		storage_obj, _, err := cosClient.List(context.Background(), tt.prefix, tt.delimiter)
		if tt.wantErr != nil {
			require.Equal(t, tt.wantErr.Error(), err.Error())
			continue
		}
		require.NoError(t, err)
		require.Contains(t, storage_obj, tt.storage_obj)
	}
}

func (cosClient *mockCosClient) DeleteObjectWithContext(ctx context.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	if *input.Bucket != cosClient.bucket {
		return &s3.DeleteObjectOutput{}, errMissingBucket
	}
	if _, ok := cosClient.data[*input.Key]; !ok {
		return &s3.DeleteObjectOutput{}, errMissingObject
	}
	delete(cosClient.data, *input.Key)

	return nil, nil
}

func (cosClient *mockCosClient) ListObjectsV2WithContext(ctx context.Context, input *s3.ListObjectsV2Input, opts ...request.Option) (*s3.ListObjectsV2Output, error) {
	if *input.Bucket != cosClient.bucket {
		return &s3.ListObjectsV2Output{}, errMissingBucket
	}
	var objects []*s3.Object
	t := time.Now()
	for object := range cosClient.data {
		objects = append(objects, &s3.Object{
			Key:          &object,
			LastModified: &t,
		})
	}

	return &s3.ListObjectsV2Output{
		Contents: objects,
	}, nil
}
