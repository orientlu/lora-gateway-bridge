package tracing

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"io"

	opentracing "github.com/opentracing/opentracing-go"
	opentracingLog "github.com/opentracing/opentracing-go/log"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

type tracingLogger struct{}

func (tl *tracingLogger) Error(msg string) {
	log.Error(msg)
}

func (tl *tracingLogger) Infof(msg string, args ...interface{}) {
	log.Debugf(msg, args...)
}

var (
	service = "Gatewayserver"
	Tracer  opentracing.Tracer
	closer  io.Closer
	logger  jaeger.Logger
)

func init() {
	logger = &tracingLogger{}
}

// InitTracing init tracing when system start
func InitTracing(serviceName string) {
	if serviceName != "" {
		service = serviceName
	}
	log.Infof("Init tracing, service name: %s", service)

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		log.Warningf("Init tracing config from env error: %v", err)
		return
	}

	Tracer, closer, err = cfg.New(service, jaegercfg.Logger(logger))
	if err != nil {
		log.Warningf("Init tracing tracer err: %v", err)
		return
	}
	opentracing.InitGlobalTracer(Tracer)
}

// DeInitTracing close tracing before system quit
func DeInitTracing() {
	if closer != nil {
		closer.Close()
	}
}

// InjectSpanContextIntoBinaryCarrier inject span into Binary carrier
func InjectSpanContextIntoBinaryCarrier(tracer opentracing.Tracer, span opentracing.Span) ([]byte, error) {
	var buf bytes.Buffer
	err := tracer.Inject(span.Context(), opentracing.Binary, &buf)
	if err != nil {
		span.LogFields(opentracingLog.String("event", "Tracer.Inject() failed"), opentracingLog.Error(err))
		buf.Reset()
	}
	return buf.Bytes(), err
}

// ExtractSpanContextFromBinaryCarrier extract the spancontext form binary carrier
func ExtractSpanContextFromBinaryCarrier(tracer opentracing.Tracer, carrier []byte) (opentracing.SpanContext, error) {
	return tracer.Extract(opentracing.Binary, bytes.NewBuffer(carrier))
}
