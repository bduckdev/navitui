run: build
	./bin/main ${ARGS}

build:
	go build -o ./bin/main ./cmd/navitui/main.go


.PHONY:run build
