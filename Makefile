# Copyright 2009 The Go Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

TARG=iconv

GOFILES=\
	reader.go

CGOFILES=\
	iconv.go\
	converter.go

ifeq ($(GOOS),windows)
CGO_LDFLAGS=-liconv
endif

# To add flags necessary for locating the library or its include files,
# set CGO_CFLAGS or CGO_LDFLAGS.  For example, to use an
# alternate installation of the library:
#	CGO_CFLAGS=-I/home/rsc/gmp32/include
#	CGO_LDFLAGS+=-L/home/rsc/gmp32/lib
# Note the += on the second line.

CLEANFILES+=sample

include $(GOROOT)/src/Make.pkg

# simple test program to test iconv conversion
sample: install sample.go
	$(GC) $@.go
	$(LD) -o $@ $@.$O
