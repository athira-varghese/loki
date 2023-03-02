package ibmcloud

import (
	"context"
	"testing"

	"github.com/IBM/ibm-cos-sdk-go/aws/request"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	"github.com/IBM/ibm-cos-sdk-go/service/s3/s3iface"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/pkg/storage/chunk/client/hedging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func Test_COSConfig(t *testing.T) {
	tests := []struct {
		name          string
		cosConfig     COSConfig
		expectedError error
	}{
		{
			"empty accessKeyID and secretAccessKey",
			COSConfig{
				BucketNames: "test",
				Endpoint:    "test",
				Region:      "dummy",
				AccessKeyID: "dummy",
			},
			errors.Wrap(errInvalidCOSHMACCredentials, "failed to build cos config"),
		},
	}
	for _, tt := range tests {
		_, err := NewCOSObjectClient(tt.cosConfig, hedging.Config{})
		require.Equal(t, tt.expectedError.Error(), err.Error())
	}
}

type mockS3Client struct {
	s3iface.S3API
	data   map[string][]byte
	bucket string
}

var (
	bucket   = "test"
	testData = map[string][]byte{
		"key-1": []byte("test data 1"),
		"key-2": []byte("test data 2"),
		"key-3": []byte("test data 3"),
	}
	errMissingBucket = errors.New("bucket not found")
	errMissingKey    = errors.New("key not found")
)

func newMockS3Client() *mockS3Client {
	return &mockS3Client{
		data:   testData,
		bucket: bucket,
	}
}

func (s3Client *mockS3Client) DeleteObjectWithContext(ctx context.Context, input *s3.DeleteObjectInput, opts ...request.Option) (*s3.DeleteObjectOutput, error) {
	if *input.Bucket != s3Client.bucket {
		return &s3.DeleteObjectOutput{}, errMissingBucket
	}
	_, ok := s3Client.data[*input.Key]

	if !ok {
		return &s3.DeleteObjectOutput{}, errMissingKey
	}

	delmarker := true
	reqcharged := ""
	versionId := "SOMEID"
	output := s3.DeleteObjectOutput{
		DeleteMarker:   &delmarker,
		RequestCharged: &reqcharged,
		VersionId:      &versionId,
	}

	return &output, nil
}

func Test_DeleteObject(t *testing.T) {
	tests := []struct {
		key       string
		wantBytes []byte
		wantErr   error
	}{
		{
			"key-1",
			[]byte("test data 1"),
			nil,
		},
		{
			"key-0",
			nil,
			errors.Wrap(errMissingKey, "failed to delete cos object"),
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
				MaxRetries: 5,
			},
		}
		cosClient, err := NewCOSObjectClient(cosConfig, hedging.Config{})
		require.NoError(t, err)
		cosClient.hedgedS3 = newMockS3Client()
		err = cosClient.DeleteObject(context.Background(), tt.key)
		if tt.wantErr != nil {
			require.Equal(t, tt.wantErr.Error(), err.Error())
			continue
		}
		require.NoError(t, err)
	}
}
