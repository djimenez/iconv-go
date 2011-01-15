include $(GOROOT)/src/Make.inc

# target package name
TARG=iconv

# regular go files
GOFILES=\
	reader.go\

# files that must be processed by cgo
CGOFILES=\
	converter.go\
	iconv.go\

include $(GOROOT)/src/Make.pkg
