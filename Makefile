all: events.go

events.go:
	../events-gen/events-gen $@
.PHONY: events.go