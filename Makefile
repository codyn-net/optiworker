ifeq ($(wildcard go.make),)
$(error Could not find go.make. Make sure to run ./configure first)
endif

include go.make

install: install-data
uninstall: uninstall-data

install-data:
	mkdir -p $(DESTDIR)$(datadir)/optiworker && cp data/example.conf $(DESTDIR)$(datadir)/optiworker/example.conf

uninstall-data:
	rm -f $(DESTDIR)$(datadir)/optiworker/example.conf

distcheck: $(TARGET)
	tar -cJf $(TARGET)-$(version).tar.xz configure.go configure $(SOURCES) Makefile data/example.conf
