LOCATION="github.com/raphaelreyna/ez-cgi"
VERSION=`git describe --tags --abbrev=0`
YEAR=`date +%Y`
VERSION_FLAG="${LOCATION}/cmd/cmd.version=${VERSION}"
YEAR_FLAG="${LOCATION}/cmd/cmd.year=${YEAR}"
LD_FLAGS="-X ${VERSION_FLAG} -X ${YEAR_FLAG}"
MANPATH=/usr/local/share/man
PREFIX=/usr/local
HERE:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

ez-cgi:
	go build -o ez-cgi -ldflags $(LD_FLAGS) .

README.md:
	cd doc/md && go run -ldflags $(LD_FLAGS) \
	. > $(HERE)/README.md

.PHONY: install
install: ez-cgi
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp $< $(DESTDIR)$(PREFIX)/bin/ez-cgi

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/ez-cgi

.PHONY: clean
clean:
	rm -f ez-cgi
