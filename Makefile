release_version:= v0.96

export GO111MODULE=on

.PHONY: bin
bin:
	go build -o bin/rsg github.com/mhausenblas/right-size-guide

.PHONY: release
release:
	curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --config .goreleaser.yml

.PHONY: publish
publish:
	git tag ${release_version}
	git push --tags