.PHONY: lint test bench install-hooks

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

bench:
	go test -bench=. -benchmem ./...

install-hooks:
	git config core.hooksPath .github/hooks
	@echo "Git hooks installed from .github/hooks/"
