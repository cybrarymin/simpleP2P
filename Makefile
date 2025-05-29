## help: print the help message
-include .envrc # -include will include .envrc but if it doesn't exist it won't return error. .envrc usually is not commited in git so to avoid pipeline failure we do this

#================================================================#
# HELPERS
#================================================================#

# always use helo as the first target. Because make command without any target will run first target defined in it. "make" will equal to "make help"
.PHONY: help # .PHONY for each target will teach make if we have a local file or directory that names help pls don't consider them and use the target we defined cause make command can't dinstingush the directory or file from targets we define inside makefile and it get's confused
help: # @ before the command will not echo the command itself when we run make <target> command
	@echo "Usage:" 
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# installing the prerequisites required by other make targets such as staticcheck command.
# after installing required binaries we will add the GO binary path to the shell $PATH
.PHONY: prerequsite
prerequsite:
	@echo "Installing Go required tools..."
	@go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Updating shell configuration..."
	@GOPATH=$$(go env GOPATH); \
	PATH_EXPORT="export PATH=\$$PATH:\$$GOPATH/bin"; \
	for RC in $$HOME/.zshrc $$HOME/.bashrc; do \
		if [ -f "$$RC" ] && ! grep -q "export PATH=.*GOPATH" "$$RC"; then \
			echo "\n# Go binaries path\n$$PATH_EXPORT" >> "$$RC"; \
		fi; \
	done; 
	@echo "Prerequisites installed successfully!"



#================================================================#
# DEVELOPMENT
#================================================================#
## build/p2p: building the application
current_time = $(shell date +"%Y-%m-%dT%H:%M:%S%z")
git_version = $(shell git describe --always --long --dirty --tags 2>/dev/null; if [[ $$? != 0 ]]; then git describe --always --dirty; fi)

Linkerflags = -s -X github.com/cybrarymin/simpleP2P/cmd.BuildTime=${current_time} -X github.com/cybrarymin/simpleP2P/cmd.Version=${git_version}
.PHONY: build/p2p
build/p2p:
	@go mod tidy
	GOOS=linux GOARCH=amd64 go build -ldflags="${Linkerflags}" -o=./bin/p2pApp-linux-amd64 ./
	GOOS=darwin GOARCH=arm64 go build -ldflags="${Linkerflags}" -o=./bin/p2pApp-darwin-arm64 ./
	go build -o=./bin/p2pApp-local-compatible -ldflags="${Linkerflags}" ./


.PHONY: build/p2p/dockerImage
## build/p2p/dockerImage: building the docker image of the the application
build/p2p/dockerImage:
	@docker build --build-arg Linkerflags="${Linkerflags}" -t "${DOCKER_IMAGENAME}":"${git_version}" ./


## run/p2p/boostrap: runs the boostrap node on port 6881 
.PHONY: run/p2p/boostrap
run/p2p/boostrap:
	@go run -race main.go  \
	--log-level=${LOGLVL} \
	--bootstrap-node=true \
	--node-address="127.0.0.1" \
	--node-port="6881"

## run/p2p/peer: runs the app as a simple ( not boostrap ) node on a random port
.PHONY: run/p2p/peer
run/p2p/peer:
	@go run -race main.go  \
	--log-level=${LOGLVL} \
	--bootstrap-node=false \
	--bootstrap-node-address="localhost:6881" \
	--node-address="127.0.0.1" \
	--node-port=`jot -r 1 6882 6889`


## proto: create the proto code and files
.PHONY: proto
proto:
#	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@protoc -I ./ --go_out=./ --go-grpc_out=./ proto/types/*.proto proto/*.proto

#================================================================#
# QUALITY CHECK , LINTING, Vendoring
#================================================================#
## audit: runs the application audit checks such as tests, lintings, staticchecks and ....
.PHONY: audit
audit: prerequsite
	@echo "Tidying and verifying golang packages and module dependencies..."
	go mod tidy
	go mod verify
	@echo "Formatting codes..."
	go fmt ./...
	@echo "Vetting codes..."
	go vet ./...
	@echo "Static Checking of the code..."
	staticcheck ./...
	@echo "Running tests..."
	go test -race -vet=off ./...

.PHONY: vendor
vendor:
	@echo "Tidying and verifying golang packages and module dependencies..."
	go mod verify
	go mod tidy
	@echo "Vendoring all golang dependency modules and packages..."
	go mod vendor