

PROG=ws-send

usage:
	@echo "usage: make [build|run]"

build b:
	go build -o $(PROG) $(PROG).go

run r:
	./$(PROG)

open o:
	open http://localhost:8080

clean:
	rm -f $(PROG)

git-update gu:
	cd ../../..; make gu
