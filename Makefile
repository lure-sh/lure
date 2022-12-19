PREFIX ?= /usr/local

lure: version.txt
	go build

clean:
	rm -f lure

install: lure
	install -Dm755 lure $(DESTDIR)$(PREFIX)/bin/lure
	install -Dm755 scripts/completion/bash $(DESTDIR)$(PREFIX)/share/bash-completion/completions/lure
	install -Dm755 scripts/completion/zsh $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_lure

uninstall:
	rm -f /usr/local/bin/lure
	
version.txt:
	go generate ./...

.PHONY: install clean uninstall