package s3

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/turbot/steampipe-plugin-sdk/v2/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin/transform"
)

func tableS3(ctx context.Context, p *plugin.Plugin) (*plugin.Table, error) {
	bucketName := "alaffia-audit-documentation-prod"
	return &plugin.Table{
		Name: bucketName,
		Description: fmt.Sprintf("A table populated with the keys and object metadata of the S3 Bucket %s", bucketName),
		List: &plugin.ListConfig{
			Hydrate: listS3Objects(bucketName),
			KeyColumns: plugin.OptionalColumns([]string{"provider_id", "icn"}),
		},
		Columns: []*plugin.Column{
			{
				Name: "provider_id", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("provider_id"), 
				Description: "A unique identifier for the healthcare provider.",
			},
			{
				Name: "icn", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("icn"), 
				Description: "A unique identifier for the medical claim.",
			},
			{
				Name: "filename", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("filename"), 
				Description: "The title of the file stored in S3.",
			},
			{
				Name: "process", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("process"), 
				Description: "The data pipeline processing task that uploaded this file to S3.",
			},
			{
				Name: "sub_process", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("sub_process"), 
				Description: "An additional processing task performed during the creation of this file.",
			},
			{
				Name: "sub_file", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("sub_file"), 
				Description: "The title of a file derived from an existing file produced by a data pipeline processing task.",
			},
			{
				Name: "etag", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("etag"), 
				Description: "The ETag of the S3 Object.",
			},
			{
				Name: "size", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("size"), 
				Description: "The memory size of the S3 Object.",
			},
			{
				Name: "s3_key", 
				Type: proto.ColumnType_STRING, 
				Transform: transform.FromField("s3_key"), 
				Description: "The unique key of the S3 Object.",
			},
			{
				Name: "tags", 
				Type: proto.ColumnType_JSON, 
				Transform: transform.FromValue(),
				Description: "The tags of the S3 Object populated by the GetObjectTagging API.",
				Hydrate: getS3ObjectTags,
			},
		},
	}, nil
}

// Implement GetObjectTagging API to hydrate the 'tags' column
func getS3ObjectTags(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	item := h.Item.(map[string]string)
	s3Key := item["s3_key"]

	client, err := S3Client(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error(err.Error())
		return nil, err
	}

	params := &s3.GetObjectTaggingInput{
		Bucket: aws.String("alaffia-audit-documentation-prod"),
		Key: aws.String(s3Key),
	}

	output, err := client.GetObjectTagging(context.TODO(), params)
	if err != nil {
		plugin.Logger(ctx).Error(err.Error(), params)
		return nil, err
	}

	tags := map[string]string{}
	for _, tag := range output.TagSet {
		tags[*tag.Key] = *tag.Value
	}

	return tags, nil
}

// Provide an S3 Key Convention as mapping of array indices to column names
// e.g. provider_id/icn/filename/process/sub_file -> [provider_id, icn, filename, process, sub_file]
func idxToColFromConfig(ctx context.Context) map[int]string {
	// TODO idxToCol := ctx.Value("idxToCol")
	idxToCol := map[int]string {
		0: "provider_id",
		1: "icn",
		2: "filename",
		3: "process",
		4: "sub_file",
	}

	return idxToCol
}

// Transform query qualifiers for key columns into an S3 Key prefix 
// for performant filtering in the ListObjectsV2 API
func columnsToS3KeyPrefix(ctx context.Context, k plugin.KeyColumnEqualsQualMap) (string) {
	idxToCol := idxToColFromConfig(ctx)

	s3KeyPrefix := ""
	for i := 0; i < len(k); i++ {
		if col, exists := idxToCol[i]; exists {
			s3KeyPrefix += k[col].GetStringValue()
			// Add a trailing slash for S3 prefix convention
			s3KeyPrefix += "/"
		} else {
			break
		}
	}

	return s3KeyPrefix
}

// Transform an S3 Key into a map[col:value], i.e. a table row
func s3KeyToRow(ctx context.Context, s3Key string) (map[string]string, error) {
	idxToCol := idxToColFromConfig(ctx)

	items := strings.Split(s3Key, "/")

	if len(items) > 5 {
		idxToCol[4] = "sub_process"
		idxToCol[5] = "sub_file"
	}

	row := map[string]string{}
	var err error
	for idx, val := range idxToCol {
		if idx < len(items) {
			row[val] = items[idx]
		}
	}

	return row, err
}

func listS3Objects(bucketName string) func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	return func(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
		// Initialize a cached S3 Client
		client, err := S3Client(ctx, d)
		if err != nil {
			plugin.Logger(ctx).Error(err.Error())
			return nil, err
		}

		// Limit the number of keys to throttle 
		// number of requests sent to GetObjectTagging API
		// when querying for tags of an object
		queryingTags := false
		for _, col := range(d.QueryContext.Columns) {
			if col == "tags" {
				queryingTags = true
				break
			}
		}
		var maxKeys int32
		if queryingTags {
			maxKeys = 100
		} else {
			maxKeys = 1000
		}
		
		// Determine if a prefix filter is possible
		// by inspecting the key column qualifiers
		var prefix *string
		s3KeyPrefix := columnsToS3KeyPrefix(ctx, d.KeyColumnQuals)

		if s3KeyPrefix == "" {
			prefix = nil
		} else {
			prefix = aws.String(s3KeyPrefix)
		}

		// Build params for ListObjects API call
		params := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
			MaxKeys: maxKeys,
			Prefix: prefix,
		}
		plugin.Logger(ctx).Debug("ListObjectsV2Input Prefix", s3KeyPrefix)
		plugin.Logger(ctx).Debug("ListObjectsV2Input MaxKeys", maxKeys)

		// Send request for s3 keys
		paginator := s3.NewListObjectsV2Paginator(client, params)
		for paginator.HasMorePages() {
			output, err := paginator.NextPage(context.TODO())
			
			if err != nil {
				plugin.Logger(ctx).Error(err.Error())
				return nil, err
			}

			// Create table row
			for _, object := range output.Contents {
				s3Key := aws.ToString(object.Key)
		
				// Get standard columns
				row := map[string]string{
					"s3_key": s3Key,
					"etag": aws.ToString(object.ETag),
					"size": strconv.FormatInt(object.Size, 10),
				}

				// Get columns from s3 key
				s3KeyRow, err := s3KeyToRow(ctx, s3Key)
				if err != nil {
					return nil, err
				}

				// Combine column sets
				for col, val := range(s3KeyRow) {
					row[col] = val
				}
	
				d.StreamListItem(ctx, row)
			}
		}

		return nil, nil
	}
}
