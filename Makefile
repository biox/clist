# clist - simple mailing list
.POSIX:

include config.mk

all: options clean clist

options:
	@echo clist build options:
	@echo "VERSION = $(VERSION)"
	@echo "PREFIX  = $(PREFIX)"

clean:
	rm -f clist

clist:
	go build -o clist -v

install: clist
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp -f clist $(DESTDIR)$(PREFIX)/bin
	chmod 755 $(DESTDIR)$(PREFIX)/bin/clist

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/clist
