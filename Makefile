source = gc-anatomy.adoc

doc:
	asciidoctor -D docs -o index.html $(source)

check:
	@sed '/----/,/----/d' < $(source) > tmp-aspell.adoc && \
	aspell \
		-d en_GB \
		--add-extra-dicts=./.aspell/common.pws \
		--add-extra-dicts=./.aspell/extra.pws \
		list < tmp-aspell.adoc | \
	tee /dev/tty | \
	wc -l | xargs -I % bash -c "test % -eq 0" || \
	rm -f tmp-aspell.adoc
	@rm -f tmp-aspell.adoc
