# standard GO make file preamble
include $(GOROOT)/src/Make.inc

# target package name
TARG=iconv

# regular go files
GOFILES=\
	reader.go\
	writer.go\

# files that must be processed by cgo
CGOFILES=\
	converter.go\
	iconv.go\

# on non glibc systems, we usually need to load the library
ifneq ($(GOOS),linux)
CGO_LDFLAGS=-liconv
endif

# standard GO make file include for packages
include $(GOROOT)/src/Make.pkg