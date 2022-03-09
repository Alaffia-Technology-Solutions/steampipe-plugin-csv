package s3

import (
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin/schema"
)

type s3Config struct {
	Paths     []string `cty:"paths"`
	Separator *string  `cty:"separator"`
	Comment   *string  `cty:"comment"`
}

var ConfigSchema = map[string]*schema.Attribute{
	"paths": {
		Type: schema.TypeList,
		Elem: &schema.Attribute{Type: schema.TypeString},
	},
	"separator": {
		Type: schema.TypeString,
	},
	"comment": {
		Type: schema.TypeString,
	},
}

func ConfigInstance() interface{} {
	return &s3Config{}
}

// GetConfig :: retrieve and cast connection config from query data
func GetConfig(connection *plugin.Connection) s3Config {
	if connection == nil || connection.Config == nil {
		return s3Config{}
	}
	config, _ := connection.Config.(s3Config)
	return config
}
