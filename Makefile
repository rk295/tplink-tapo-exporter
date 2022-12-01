build: *.go exporter/*.go go.mod go.sum
	go build .

clean:
	rm tplink-tapo-exporter