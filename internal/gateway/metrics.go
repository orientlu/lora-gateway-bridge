package gateway

import (
	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gatewayEventCounter      func(string)
	gatewayHandleTimer       func(string, func() error) error
	gatewayConfigHandleTimer func(func() error) error
)

func init() {
	ec := metrics.MustRegisterNewCounter(
		"gateway_event",
		"Per event type counter.",
		[]string{"event"},
	)

	ht := metrics.MustRegisterNewTimerWithError(
		"gateway_udp_handle",
		"Per messate-type handle duration tracking.",
		[]string{"type"},
	)

	ch := metrics.MustRegisterNewTimerWithError(
		"gateway_config_handle",
		"Tracks the duration of configuration handling.",
		[]string{},
	)

	gatewayEventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}

	gatewayHandleTimer = func(mType string, f func() error) error {
		return ht(prometheus.Labels{"type": mType}, f)
	}

	gatewayConfigHandleTimer = func(f func() error) error {
		return ch(prometheus.Labels{}, f)
	}
}
