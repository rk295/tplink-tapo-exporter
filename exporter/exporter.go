package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rk295/tapo-go"
	log "github.com/sirupsen/logrus"
)

type Exporter struct {
	target     string
	tapoDevice *tapo.Device

	metricsUp,
	metricsMetadata,
	metricsRelayState,
	metricsOnTime,
	metricsRssi,
	metricsCurrent *prometheus.Desc
}

type ExporterTarget struct {
	Host       string
	TapoDevice *tapo.Device
}

func NewExporter(t *ExporterTarget) *Exporter {
	var (
		constLabels = prometheus.Labels{}
		labelNames  = []string{"nickname", "id"}
	)

	e := &Exporter{
		target:     t.Host,
		tapoDevice: t.TapoDevice,
		metricsUp: prometheus.NewDesc("tapo_online",
			"Device online.",
			nil, constLabels,
		),

		metricsMetadata: prometheus.NewDesc("tapo_metadata",
			"Device metadata.",
			[]string{
				"nickname", "hw_ver", "sw_ver", "model",
			}, constLabels,
		),

		metricsRelayState: prometheus.NewDesc("tapo_relay_state",
			"Relay state (switch on/off).",
			labelNames, constLabels,
		),
		metricsOnTime: prometheus.NewDesc("tapo_on_time",
			"Time in seconds since online.",
			labelNames, constLabels),
		metricsRssi: prometheus.NewDesc("tapo_rssi",
			"Wifi received signal strength indicator.",
			labelNames, constLabels),

		metricsCurrent: prometheus.NewDesc("tapo_current",
			"Current flowing through device in milliwatts (mW).",
			labelNames, constLabels),
	}
	return e
}

func (k *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- k.metricsCurrent
	ch <- k.metricsMetadata
	ch <- k.metricsOnTime
	ch <- k.metricsRelayState
	ch <- k.metricsRssi
	ch <- k.metricsUp

}

func (k *Exporter) Collect(ch chan<- prometheus.Metric) {

	err := k.tapoDevice.Login()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(k.metricsUp, prometheus.GaugeValue,
			0)
		log.Errorln("error collecting", k.target, ":", err)
		return
	}

	i, err := k.tapoDevice.GetDeviceInfo()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(k.metricsUp, prometheus.GaugeValue,
			0)
		log.Errorln("error collecting", k.target, ":", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(k.metricsMetadata, prometheus.GaugeValue,
		1, i.Nickname, i.HWVersion, i.FWVersion, i.Model)

	relayState := 0
	if i.DeviceON {
		relayState = 1
	}
	ch <- prometheus.MustNewConstMetric(k.metricsRelayState, prometheus.GaugeValue,
		float64(relayState), i.Nickname, i.DeviceID)

	ch <- prometheus.MustNewConstMetric(k.metricsOnTime, prometheus.CounterValue,
		float64(i.OnTime), i.Nickname, i.DeviceID)

	ch <- prometheus.MustNewConstMetric(k.metricsRssi, prometheus.GaugeValue,
		float64(i.RSSI), i.Nickname, i.DeviceID)

	if i.EmeterSupported() {

		e, err := k.tapoDevice.GetEnergyUsage()
		if err != nil {
			ch <- prometheus.MustNewConstMetric(k.metricsUp, prometheus.GaugeValue,
				0)
			log.Errorln("error collecting", k.target, ":", err)
			return
		}

		ch <- prometheus.MustNewConstMetric(k.metricsCurrent, prometheus.GaugeValue,
			float64(e.CurrentPower), i.Nickname, i.DeviceID)
	}

	ch <- prometheus.MustNewConstMetric(k.metricsUp, prometheus.GaugeValue,
		1)
}
