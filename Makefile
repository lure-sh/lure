lure: version.txt
	go build

clean:
	rm -f lure

install: lure
	sudo install -Dm755 lure /usr/local/bin/lure

uninstall:
	rm -f /usr/local/bin/lure
	
version.txt:
	go generate

.PHONY: install clean uninstall