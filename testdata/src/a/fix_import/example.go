package fix_import

import (
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"context"
)

type telemetry struct {
	TraceLogKey string
}

func SomeFunc(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	log := zapr.NewLogger(zap.L()).WithValues("eventType", eventType, "deliverID", deliveryID)  // want `missing traceId in logging keys`
	log = log.WithValues("eventType", "hello")
	log.Info("Tracing")
	return nil
}

func SomeFunc1(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	telemetryInstance := telemetry{
		TraceLogKey: "dummyTrace",
	}
	log := zapr.NewLogger(zap.L()).WithValues(telemetryInstance.TraceLogKey, "someValue") // cannot be detected
	log = log.WithValues("eventType", "hello")
	log.Info("Tracing")
	return nil
}

func SomeFunc2(eventType, deliveryID string, payload []byte) error {
	log := zapr.NewLogger(zap.L()).WithValues("eventType", eventType, "deliverID", deliveryID) // cannot be detected
	log = log.WithValues("eventType", "hello")
	log.Info("Tracing")
	return nil
}