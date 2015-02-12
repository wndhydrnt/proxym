TRAVIS_TAG ?= build

test:
	go test -v ./...

release:
	go get github.com/mitchellh/gox
	gox -build-toolchain -os="linux"
	gox -os="linux"

package: release fpm package_deb_amd64 package_deb_i386

package_deb_i386:
	mkdir -p ./package/deb/usr/sbin
	cp -f proxym_linux_386 ./package/deb/usr/sbin/proxym
	bundle exec fpm --version $(TRAVIS_TAG) --architecture i386 --config-files etc/ --deb-upstart package/proxym -n proxym -C ./package/deb -t deb -s dir .

package_deb_amd64:
	mkdir -p ./package/deb/usr/sbin
	cp -f proxym_linux_amd64 ./package/deb/usr/sbin/proxym
	bundle exec fpm --version $(TRAVIS_TAG) --architecture x86_64 --config-files etc/ --deb-upstart package/proxym -n proxym -C ./package/deb -t deb -s dir .

fpm:
	bundle install

.PHONY: clean
clean:
	rm -rf proxym_*
