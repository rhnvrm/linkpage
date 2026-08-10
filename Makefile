# linkpage - build the full app.
# `make` (default target) builds everything: frontend assets via esgun
# (pinned as a Go tool dependency in go.mod), then the Go binary. The built
# assets land in static/app/dist and are embedded into the binary.
ESGUN ?= go tool esgun
FRONTEND = static/app
BINARY = linkpage

.PHONY: build frontend clean

build: frontend
	go build -o $(BINARY) .

frontend:
	$(ESGUN) deps --dir $(FRONTEND)
	$(ESGUN) build --dir $(FRONTEND)

clean:
	rm -f $(BINARY)
