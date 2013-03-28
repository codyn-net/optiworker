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
	@echo "[GEN] $(TARGET)-$(version).tar.gz"; \
	rm -rf $(TARGET)-$(version); \
	mkdir $(TARGET)-$(version); \
	mkdir $(TARGET)-$(version)/build; \
	mkdir $(TARGET)-$(version)/data; \
	cp build/configure.go $(TARGET)-$(version)/build/; \
	cp data/example.conf $(TARGET)-$(version)/data/; \
	cp configure Makefile $(sort $(SOURCES)) $(TARGET)-$(version)/; \
	tar -cjf $(TARGET)-$(version).tar.bz2 $(TARGET)-$(version); \
	rm -rf $(TARGET)-$(version)

debian-test: distcheck
	tar -xjf $(TARGET)-$(version).tar.bz2; \
	tar -czf $(TARGET)_$(version).orig.tar.gz $(TARGET)-$(version); \
	cp -r debian $(TARGET)-$(version); \
	(cd $(TARGET)-$(version) && dpkg-buildpackage)
