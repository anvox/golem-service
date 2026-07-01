build_config_command:
	go build -o ./dist/golem-config ./cmd/config/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-config-darwin-amd64 ./cmd/config/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-config-linux-amd64 ./cmd/config/*.go
	chmod a+x ./dist/golem-config*
build_ps_command:
	go build -o ./dist/golem-ps ./cmd/ps/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-ps-darwin-amd64 ./cmd/ps/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-ps-linux-amd64 ./cmd/ps/*.go
	chmod a+x ./dist/golem-ps*
build_entrypoint:
	go build -o ./dist/golem-entrypoint ./cmd/entrypoint/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-entrypoint-darwin-amd64 ./cmd/entrypoint/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-entrypoint-linux-amd64 ./cmd/entrypoint/*.go
	chmod a+x ./dist/golem-entrypoint*
release_entrypoint:
	go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-entrypoint ./cmd/entrypoint/*.go
	env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-entrypoint-darwin-amd64 ./cmd/entrypoint/*.go
	env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ./dist/release/golem-entrypoint-linux-amd64 ./cmd/entrypoint/*.go
	chmod a+x ./dist/release/golem-entrypoint*
build_exec:
	go build -o ./dist/golem-exec-remote ./cmd/exec-remote/*.go
	env GOOS=darwin GOARCH=amd64 go build -o ./dist/golem-exec-remote-darwin-amd64 ./cmd/exec-remote/*.go
	env GOOS=linux GOARCH=amd64 go build -o ./dist/golem-exec-remote-linux-amd64 ./cmd/exec-remote/*.go
	chmod a+x ./dist/golem-exec-remote*
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
