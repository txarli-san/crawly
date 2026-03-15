NAME    := crawly
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST    := dist

.PHONY: linux windows clean dist

linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST)/linux/$(NAME) .
	cp -r assets $(DIST)/linux/assets

windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o $(DIST)/windows/$(NAME).exe .
	cp -r assets $(DIST)/windows/assets

dist: linux windows
	cd $(DIST)/linux && tar czf ../$(NAME)-$(VERSION)-linux-amd64.tar.gz $(NAME) assets
	cd $(DIST)/windows && tar czf ../$(NAME)-$(VERSION)-windows-amd64.tar.gz $(NAME).exe assets
	@ls -lh $(DIST)/*.tar.gz

clean:
	rm -rf $(DIST)
