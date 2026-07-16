PROJECT?=geo
APP?=suggest
PORT?=80
PORT_APP?=7784

CONTAINER_IMAGE?=$(PROJECT)/${APP}
RELEASE?=0.0.1
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}
	goimports -l -w ${GOFILES_NOVENDOR}

checkstyle:
	golangci-lint run --timeout=10m -v ./...

clean:
	rm -f bin/${APP}

gin:
	GO111MODULE=off go get github.com/codegangsta/gin

init: gin
	@echo "ready"

gorun: clean
	go build -o bin/${APP} -tags "dev load_envs" ./cmd/ && bin/${APP}

watcher: gin
	gin --build cmd/ --logPrefix watcher --immediate --buildArgs "-tags 'dev load_envs'" run

container:
	docker build -t $(CONTAINER_IMAGE):$(RELEASE) .

run: container
	docker stop $(CONTAINER_IMAGE):$(RELEASE) || true && docker rm $(CONTAINER_IMAGE):$(RELEASE) || true
	docker run --name ${APP} -p ${PORT}:${PORT_APP} --rm \
		-e "PORT=${PORT}" \
		--env-file .env  \
		$(CONTAINER_IMAGE):$(RELEASE)

test:
	go test -tags="testing" -v -race -cover -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out

codegen:
	go run indexer/codegen/main.go
	go fmt ./...

mocks:
	mockgen -destination=indexer/factory/mock/mocks.go -source=indexer/factory/interfaces.go -package=mock