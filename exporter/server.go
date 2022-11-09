package exporter

import (
	"net/http"
	"os"

	"github.com/rk295/tapo-go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HttpServer struct {
	mux          *http.ServeMux
	tapoEmail    string
	tapoPassword string
}

func NewHttpServer() *HttpServer {
	s := &HttpServer{
		mux:          http.NewServeMux(),
		tapoEmail:    os.Getenv(tapo.TapoEmailEnvName),
		tapoPassword: os.Getenv(tapo.TapoPasswordEnvName),
	}
	s.mux.HandleFunc("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP)

	s.mux.HandleFunc("/scrape", s.ScrapeHandler)
	return s
}

func (s *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

//https://github.com/oliver006/redis_exporter/blob/master/exporter.go
func (s *HttpServer) ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", 400)
		//e.targetScrapeRequestErrors.Inc()
		return
	}

	registry := prometheus.NewRegistry()
	e := NewExporter(&ExporterTarget{
		Host:       target,
		TapoDevice: tapo.New(target, s.tapoEmail, s.tapoPassword),
	})
	registry.MustRegister(e)

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
