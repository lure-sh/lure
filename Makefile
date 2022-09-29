lure:
	go build

clean:
	rm -f lure

install: lure
	sudo install -Dm755 lure /usr/local/bin/lure

uninstall:
	rm -f /usr/local/bin/lure

.PHONY: install clean uninstall