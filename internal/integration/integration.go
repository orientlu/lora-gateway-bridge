package integration

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/integration/mqtt"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

// Event types.
const (
	EventUp    = "up"
	EventStats = "stats"
	EventAck   = "ack"
)

// Notify types.
const (
	NotifyMac = "mac"
)

var integration Integration

func Setup(conf config.Config) error {
	var err error
	integration, err = mqtt.NewBackend(conf)
	if err != nil {
		return errors.Wrap(err, "setup mqtt integration error")
	}

	return nil
}

// GetIntegration returns the integration.
func GetIntegration() Integration {
	return integration
}

type Integration interface {
	// SubscribeGateway creates a subscription for the given gateway ID.
	SubscribeGateway(lorawan.EUI64) error

	// UnsubscribeGateway removes the subscription for the given gateway ID.
	UnsubscribeGateway(lorawan.EUI64) error

	// PublishEvent publishes the given event.
	PublishEvent(context.Context, lorawan.EUI64, string, proto.Message) error

	// PublishNotifyEvent publishes the given notify event.
	PublishNotifyEvent(string, proto.Message) error

	// GetDownlinkFrameChan returns the channel for downlink frames.
	GetDownlinkFrameChan() chan gw.DownlinkFrame

	// GetGatewayConfigurationChan returns the channel for gateway configuration.
	GetGatewayConfigurationChan() chan gw.GatewayConfiguration

	// Close closes the integration.
	Close() error
}
