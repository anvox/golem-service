build_config_command:
	go build -o ./dist/golem-service-config ./cmd/config/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-service-config-darwin-amd64 ./cmd/config/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-service-config-linux-amd64 ./cmd/config/*.go
	chmod a+x ./dist/golem-service-config*
build_ps_command:
	go build -o ./dist/golem-service-ps ./cmd/ps/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-service-ps-darwin-amd64 ./cmd/ps/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-service-ps-linux-amd64 ./cmd/ps/*.go
	chmod a+x ./dist/golem-service-ps*
build_entrypoint:
	go build -o ./dist/golem-service-entrypoint ./cmd/entrypoint/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-service-entrypoint-darwin-amd64 ./cmd/entrypoint/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-service-entrypoint-linux-amd64 ./cmd/entrypoint/*.go
	chmod a+x ./dist/golem-service-entrypoint*
release_entrypoint:
	go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-service-entrypoint ./cmd/entrypoint/*.go
	env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-service-entrypoint-darwin-amd64 ./cmd/entrypoint/*.go
	env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-service-entrypoint-linux-amd64 ./cmd/entrypoint/*.go
	chmod a+x ./dist/release/golem-service-entrypoint*
build_exec:
	go build -o ./dist/golem-service-exec-remote ./cmd/exec-remote/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-service-exec-remote-darwin-amd64 ./cmd/exec-remote/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-service-exec-remote-linux-amd64 ./cmd/exec-remote/*.go
	chmod a+x ./dist/golem-service-exec-remote*
build_deploy_command:
	go build -o ./dist/golem-deploy ./cmd/deploy/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-deploy-darwin-amd64 ./cmd/deploy/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-deploy-linux-amd64 ./cmd/deploy/*.go
	chmod a+x ./dist/golem-deploy*
build: build_config_command build_ps_command build_entrypoint build_exec build_deploy_command
deploy: build
	echo "Please push to github release manually"
clean:
	rm -rf ./dist/*
