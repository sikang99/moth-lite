#
# Makefile
#
PROG=exip
usage:
	@echo "usage: make [build|run]"

edit e:
	vi main.go

build b:
	go build -o $(PROG) *.go

run r:
	go run main.go

clean:
	rm -f $(PROG)

git-update gu:
	cd ..; make gu


