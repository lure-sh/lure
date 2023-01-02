PREFIX ?= /usr/local

lure: version.txt
	go build

clean:
	rm -f lure

install: lure installmisc
	install -Dm755 lure $(DESTDIR)$(PREFIX)/bin/lure

installmisc:
	install -Dm755 scripts/completion/bash $(DESTDIR)$(PREFIX)/share/bash-completion/completions/lure
	install -Dm755 scripts/completion/zsh $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_lure

uninstall:
	rm -f /usr/local/bin/lure
	
internal/config/version.txt:
	go generate ./internal/config

.PHONY: install clean uninstall