package mqtt

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/integration/mqtt/auth"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

// Backend implements a MQTT backend.
type Backend struct {
	sync.RWMutex

	auth                     auth.Authentication
	conn                     paho.Client
	closed                   bool
	clientOpts               *paho.ClientOptions
	downlinkFrameChan        chan gw.DownlinkFrame
	gatewayConfigurationChan chan gw.GatewayConfiguration
	gateways                 map[lorawan.EUI64]struct{}

	qos                  uint8
	eventTopicTemplate   *template.Template
	NotifyTopicTemplate  *template.Template
	commandTopicTemplate *template.Template

	marshal   func(msg proto.Message) ([]byte, error)
	unmarshal func(b []byte, msg proto.Message) error
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	var err error

	b := Backend{
		qos:                      conf.Integration.MQTT.Auth.Generic.QOS,
		clientOpts:               paho.NewClientOptions(),
		downlinkFrameChan:        make(chan gw.DownlinkFrame),
		gatewayConfigurationChan: make(chan gw.GatewayConfiguration),
		gateways:                 make(map[lorawan.EUI64]struct{}),
	}

	switch conf.Integration.MQTT.Auth.Type {
	case "generic":
		b.auth, err = auth.NewGenericAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integation/mqtt: new generic authentication error")
		}
	case "gcp_cloud_iot_core":
		b.auth, err = auth.NewGCPCloudIoTCoreAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integration/mqtt: new GCP Cloud IoT Core authentication error")
		}

		conf.Integration.MQTT.EventTopicTemplate = "/devices/gw-{{ .GatewayID }}/events/{{ .EventType }}"
		conf.Integration.MQTT.NotifyTopicTemplate = "/devices/notify/{{ .EventType }}"
		conf.Integration.MQTT.CommandTopicTemplate = "/devices/gw-{{ .GatewayID }}/commands/#"
	case "azure_iot_hub":
		b.auth, err = auth.NewAzureIoTHubAuthentication(conf)
		if err != nil {
			return nil, errors.Wrap(err, "integration/mqtt: new azure iot hub authentication error")
		}

		conf.Integration.MQTT.EventTopicTemplate = "devices/{{ .GatewayID }}/messages/events/{{ .EventType }}"
		conf.Integration.MQTT.NotifyTopicTemplate = "/devices/notify/{{ .EventType }}"
		conf.Integration.MQTT.CommandTopicTemplate = "devices/{{ .GatewayID }}/messages/devicebound/#"
	default:
		return nil, fmt.Errorf("integration/mqtt: unknown auth type: %s", conf.Integration.MQTT.Auth.Type)
	}

	switch conf.Integration.Marshaler {
	case "json":
		b.marshal = func(msg proto.Message) ([]byte, error) {
			marshaler := &jsonpb.Marshaler{
				EnumsAsInts:  false,
				EmitDefaults: true,
			}
			str, err := marshaler.MarshalToString(msg)
			return []byte(str), err
		}

		b.unmarshal = func(b []byte, msg proto.Message) error {
			unmarshaler := &jsonpb.Unmarshaler{
				AllowUnknownFields: true, // we don't want to fail on unknown fields
			}
			return unmarshaler.Unmarshal(bytes.NewReader(b), msg)
		}
	case "protobuf":
		b.marshal = func(msg proto.Message) ([]byte, error) {
			return proto.Marshal(msg)
		}

		b.unmarshal = func(b []byte, msg proto.Message) error {
			return proto.Unmarshal(b, msg)
		}
	default:
		return nil, fmt.Errorf("integration/mqtt: unknown marshaler: %s", conf.Integration.Marshaler)
	}

	b.eventTopicTemplate, err = template.New("event").Parse(conf.Integration.MQTT.EventTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "integration/mqtt: parse event-topic template error")
	}

	b.NotifyTopicTemplate, err = template.New("event").Parse(conf.Integration.MQTT.NotifyTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "integration/mqtt: parse notify-topic template error")
	}

	b.commandTopicTemplate, err = template.New("event").Parse(conf.Integration.MQTT.CommandTopicTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "integration/mqtt: parse event-topic template error")
	}

	b.clientOpts.SetProtocolVersion(4)
	b.clientOpts.SetAutoReconnect(false)
	b.clientOpts.SetOnConnectHandler(b.onConnected)
	b.clientOpts.SetConnectionLostHandler(b.onConnectionLost)

	if err = b.auth.Init(b.clientOpts); err != nil {
		return nil, errors.Wrap(err, "mqtt: init authentication error")
	}

	b.connectLoop()
	go b.reconnectLoop()

	return &b, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.Lock()
	b.closed = true
	b.Unlock()

	b.conn.Disconnect(250)
	return nil
}

// GetDownlinkFrameChan returns the downlink frame channel.
func (b *Backend) GetDownlinkFrameChan() chan gw.DownlinkFrame {
	return b.downlinkFrameChan
}

// GetGatewayConfigurationChan returns the gateway configuration channel.
func (b *Backend) GetGatewayConfigurationChan() chan gw.GatewayConfiguration {
	return b.gatewayConfigurationChan
}

// SubscribeGateway subscribes a gateway to its topics.
func (b *Backend) SubscribeGateway(gatewayID lorawan.EUI64) error {
	b.Lock()
	defer b.Unlock()

	if err := b.subscribeGateway(gatewayID); err != nil {
		return err
	}

	b.gateways[gatewayID] = struct{}{}
	return nil
}

func (b *Backend) subscribeGateway(gatewayID lorawan.EUI64) error {
	topic := bytes.NewBuffer(nil)
	if err := b.commandTopicTemplate.Execute(topic, struct{ GatewayID lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute command topic template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.qos,
	}).Info("integration/mqtt: subscribing to topic")

	err := mqttSubscribeTimer(func() error {
		if token := b.conn.Subscribe(topic.String(), b.qos, b.handleCommand); token.Wait() && token.Error() != nil {
			return errors.Wrap(token.Error(), "subscribe topic error")
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// UnsubscribeGateway unsubscribes the gateway from its topics.
func (b *Backend) UnsubscribeGateway(gatewayID lorawan.EUI64) error {
	b.Lock()
	defer b.Unlock()

	topic := bytes.NewBuffer(nil)
	if err := b.commandTopicTemplate.Execute(topic, struct{ GatewayID lorawan.EUI64 }{gatewayID}); err != nil {
		return errors.Wrap(err, "execute command topic template error")
	}
	log.WithFields(log.Fields{
		"topic": topic.String(),
	}).Info("integration/mqtt: unsubscribe topic")

	err := mqttUnsubscribeTimer(func() error {
		if token := b.conn.Unsubscribe(topic.String()); token.Wait() && token.Error() != nil {
			return errors.Wrap(token.Error(), "unsubscribe topic error")
		}
		return nil
	})
	if err != nil {
		return err
	}

	delete(b.gateways, gatewayID)
	return nil
}

// PublishEvent publishes the given event.
func (b *Backend) PublishEvent(gatewayID lorawan.EUI64, event string, v proto.Message) error {
	return mqttPublishTimer(event, func() error {
		return b.publish(gatewayID, event, v)
	})
}

// PublishNotifyEvent publishes the given notify event.
func (b *Backend) PublishNotifyEvent(event string, v proto.Message) error {
	return mqttPublishTimer(event, func() error {
		return b.publishNotify(event, v)
	})
}

func (b *Backend) connect() error {
	b.Lock()
	defer b.Unlock()

	if err := b.auth.Update(b.clientOpts); err != nil {
		return errors.Wrap(err, "integration/mqtt: update authentication error")
	}

	b.conn = paho.NewClient(b.clientOpts)

	return mqttConnectTimer(func() error {
		if token := b.conn.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	})
}

// connectLoop blocks until the client is connected
func (b *Backend) connectLoop() {
	for {
		if err := b.connect(); err != nil {
			log.WithError(err).Error("integration/mqtt: connection error")
			time.Sleep(time.Second * 2)

		} else {
			break
		}
	}
}

func (b *Backend) disconnect() error {
	mqttConnectionCounter("disconnect")

	b.Lock()
	defer b.Unlock()

	b.conn.Disconnect(250)
	return nil
}

func (b *Backend) reconnectLoop() {
	if b.auth.ReconnectAfter() > 0 {
		for {
			if b.closed {
				break
			}
			time.Sleep(b.auth.ReconnectAfter())
			log.Info("mqtt: re-connect triggered")

			mqttConnectionCounter("reconnect")

			b.disconnect()
			b.connectLoop()
		}
	}
}

func (b *Backend) onConnected(c paho.Client) {
	mqttConnectionCounter("connected")

	b.RLock()
	defer b.RUnlock()

	log.Info("integration/mqtt: connected to mqtt broker")

	for gatewayID := range b.gateways {
		for {
			if err := b.subscribeGateway(gatewayID); err != nil {
				log.WithError(err).WithField("gateway_id", gatewayID).Error("integration/mqtt: subscribe gateway error")
				time.Sleep(time.Second)
				continue
			}

			break
		}
	}
}

func (b *Backend) onConnectionLost(c paho.Client, err error) {
	mqttConnectionCounter("lost")
	log.WithError(err).Error("mqtt: connection error")
	b.connectLoop()
}

func (b *Backend) handleDownlinkFrame(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("integration/mqtt: downlink frame received")

	var downlinkFrame gw.DownlinkFrame
	if err := b.unmarshal(msg.Payload(), &downlinkFrame); err != nil {
		log.WithError(err).Error("integration/mqtt: unmarshal downlink frame error")
		return
	}

	b.downlinkFrameChan <- downlinkFrame
}

func (b *Backend) handleGatewayConfiguration(c paho.Client, msg paho.Message) {
	log.WithFields(log.Fields{
		"topic": msg.Topic(),
	}).Info("integration/mqtt: gateway configuration received")

	var gatewayConfig gw.GatewayConfiguration
	if err := b.unmarshal(msg.Payload(), &gatewayConfig); err != nil {
		log.WithError(err).Error("integration/mqtt: unmarshal gateway configuration error")
		return
	}

	b.gatewayConfigurationChan <- gatewayConfig
}

func (b *Backend) handleCommand(c paho.Client, msg paho.Message) {
	if strings.HasSuffix(msg.Topic(), "down") || strings.Contains(msg.Topic(), "command=down") {
		mqttCommandCounter("down")
		b.handleDownlinkFrame(c, msg)
	} else if strings.HasSuffix(msg.Topic(), "config") || strings.Contains(msg.Topic(), "command=config") {
		mqttCommandCounter("config")
		b.handleGatewayConfiguration(c, msg)
	} else {
		log.WithFields(log.Fields{
			"topic": msg.Topic(),
		}).Warning("integration/mqtt: unexpected command received")
	}
}

func (b *Backend) publish(gatewayID lorawan.EUI64, event string, msg proto.Message) error {
	topic := bytes.NewBuffer(nil)
	if err := b.eventTopicTemplate.Execute(topic, struct {
		GatewayID lorawan.EUI64
		EventType string
	}{gatewayID, event}); err != nil {
		return errors.Wrap(err, "execute event template error")
	}

	bytes, err := b.marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message error")
	}

	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.qos,
		"event": event,
	}).Info("integration/mqtt: publishing event")
	if token := b.conn.Publish(topic.String(), b.qos, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *Backend) publishNotify(event string, msg proto.Message) error {
	topic := bytes.NewBuffer(nil)
	if err := b.NotifyTopicTemplate.Execute(topic, struct {
		NotifyType string
	}{event}); err != nil {
		return errors.Wrap(err, "execute notify event template error")
	}

	bytes, err := b.marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message error")
	}

	log.WithFields(log.Fields{
		"topic": topic.String(),
		"qos":   b.qos,
		"event": event,
	}).Info("integration/mqtt: publishing notify event")
	if token := b.conn.Publish(topic.String(), b.qos, false, bytes); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}
