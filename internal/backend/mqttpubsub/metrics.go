package mqttpubsub

import (
	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mqttPublishTimer func(string, func() error) error
	mqttHandleTimer  func(string, func() error) error
	mqttEventCounter func(string)
)

func init() {
	pt := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_publish",
		"Per message-type publish duration tracking.",
		[]string{"type"},
	)

	ht := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_handle",
		"Per message-type handle duration tracking (note 'handled' means it is internally added to the queue).",
		[]string{"type"},
	)

	ec := metrics.MustRegisterNewCounter(
		"backend_mqtt_event",
		"Per event type counter.",
		[]string{"event"},
	)

	mqttPublishTimer = func(mType string, f func() error) error {
		return pt(prometheus.Labels{"type": mType}, f)
	}

	mqttHandleTimer = func(mType string, f func() error) error {
		return ht(prometheus.Labels{"type": mType}, f)
	}

	mqttEventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}
}
