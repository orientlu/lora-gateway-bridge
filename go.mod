module github.com/brocaar/lora-gateway-bridge

go 1.12

require (
	github.com/brocaar/loraserver v2.5.0+incompatible
	github.com/brocaar/lorawan v0.0.0-20190402092148-5bca41b178e9
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/golang/protobuf v1.3.1
	github.com/goreleaser/goreleaser v0.106.0
	github.com/gorilla/websocket v1.4.0
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.4
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
)

replace github.com/brocaar/loraserver => git.code.oa.com/loraiot/networkserver v0.0.1
