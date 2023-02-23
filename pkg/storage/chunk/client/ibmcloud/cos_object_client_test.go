package ibmcloud

import (
	"testing"

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
