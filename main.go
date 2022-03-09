package main

import (
	"github.com/Alaffia-Technology-Solutions/steampipe-plugin-s3/s3"
	"github.com/turbot/steampipe-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{PluginFunc: s3.Plugin})
}
