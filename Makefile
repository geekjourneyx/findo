VERSION ?= 1.2.1
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint version-check release-check smoke-bocha smoke-volcengine smoke-zhihu

build:
	go build -buildvcs=false -trimpath -ldflags="$(LDFLAGS)" -o findo ./cmd/findo

test:
	CGO_ENABLED=0 GOFLAGS="-buildvcs=false" go test -count=1 ./...

lint:
	test -z "$$(gofmt -l .)"
	GOFLAGS="-buildvcs=false" go vet ./...
	GOFLAGS="-buildvcs=false" go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run

version-check:
	VERSION=$(VERSION) bash scripts/check-version.sh

release-check:
	VERSION=$(VERSION) bash scripts/release-check.sh

smoke-bocha:
	BOCHA_API_KEY=$${BOCHA_API_KEY} go run ./cmd/findo bocha "瑞幸咖啡 2026 门店数" --json

smoke-volcengine:
	ARK_API_KEY=$${ARK_API_KEY} go run ./cmd/findo volc "瑞幸咖啡 2026 门店数是否可信" --json

smoke-zhihu:
	ZHIHU_ACCESS_SECRET=$${ZHIHU_ACCESS_SECRET} go run ./cmd/findo zhihu "AI 搜索" --json
