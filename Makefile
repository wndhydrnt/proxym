test:
	go test -v ./...

release:
	go get github.com/mitchellh/gox
	gox -build-toolchain
	gox -os="linux"
