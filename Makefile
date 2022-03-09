install:
	go build -o ~/.steampipe/plugins/hub.steampipe.io/plugins/Alaffia-Technology-Solutions/s3@latest/steampipe-plugin-s3.plugin *.go
build:
	go build -o ./build/${GOOS}/${GOARCH}/steampipe-plugin-s3.plugin *.go
