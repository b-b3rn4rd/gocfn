ci: install lint test
install:
	go get -t ./...
	go get -u gopkg.in/alecthomas/gometalinter.v2
	gometalinter.v2 --install
	go get github.com/axw/gocov/...
	go get github.com/AlekSi/gocov-xml
	go get github.com/jstemmer/go-junit-report
lint:
	gometalinter.v2 ./... | grep -v -E "(should have comment or be unexported|comment on exported method)" || true
	gometalinter.v2 ./... --errors
test:
	go test -v ./...
release:
	goreleaser --rm-dist --config=.goreleaser.yml


