.POSIX:

VERSION=0.1.0

PREFIX?=/usr/local
_INSTDIR=$(DESTDIR)$(PREFIX)
BINDIR?=$(_INSTDIR)/bin
GO?=go
GOFLAGS?=

GOSRC!=find . -name '*.go'
GOSRC+=go.mod go.sum

clist: $(GOSRC)
	$(GO) build $(GOFLAGS) \
		-ldflags "-X main.Prefix=$(PREFIX) \
		-X main.ShareDir=$(SHAREDIR) \
		-X main.Version=$(VERSION)" \
		-o $@

all: clist

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

clean:
	$(RM) aerc

.DEFAULT_GOAL := all
