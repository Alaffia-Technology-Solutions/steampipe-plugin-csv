package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin"
)

// S3Client returns the service connection for AWS S3 service
func S3Client(ctx context.Context, d *plugin.QueryData) (*s3.Client, error) {

	// have we already created and cached the service?
	serviceCacheKey := "s3-service"
	if cachedData, ok := d.ConnectionManager.Cache.Get(serviceCacheKey); ok {
		return cachedData.(*s3.Client), nil
	}
	// so it was not in cache - create service
	// Load the Shared AWS Configuration (~/.aws/config)
	// enable experimential adaptive retry mode 
	// see https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws#RetryMode
	cfg, err := config.LoadDefaultConfig(
		context.TODO(), 
		config.WithRetryMode(aws.RetryModeAdaptive),
		config.WithRetryMaxAttempts(20),
	)
	
	if err != nil {
		plugin.Logger(ctx).Error(err.Error())
		return nil, err
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)
	d.ConnectionManager.Cache.Set(serviceCacheKey, client)

	return client, nil
}