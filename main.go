package main

import (
	"flag"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/rk295/tplink-tapo-exporter/exporter"
)

func main() {
	var metricsAddr = flag.String("metrics.listen-addr", ":9235", "listen address for tplink-tapo exporter")

	flag.Parse()
	s := exporter.NewHttpServer()
	log.Infof("Accepting Prometheus Requests on %s", *metricsAddr)
	log.Fatal(http.ListenAndServe(*metricsAddr, s))
}
