source = gc-anatomy.adoc

doc: relocate
	./relocate < $(source) | asciidoctor -D docs -o index.html -

relocate: relocate.go
	go build -o $@ $<

check:
	@sed '/----/,/----/d' < $(source) > tmp-aspell.adoc && \
	aspell \
		-d en_GB \
		--add-extra-dicts=./.aspell/common.pws \
		--add-extra-dicts=./.aspell/extra.pws \
		list < tmp-aspell.adoc | \
	tee /dev/tty | \
	wc -l | xargs -I % bash -c "test % -eq 0"
	@rm -f tmp-aspell.adoc

checkout-go:
	git clone https://github.com/golang/go.git

output:
	cd inlining/not-inlined && go tool compile -m=2 not-inlined.go &> not-inlined.output
	cd inlining/inlined && go tool compile -m inlined.go &> inlined.output
