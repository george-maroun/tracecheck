package fix_import

import (
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"context"
)

func SomeFunc(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	log := zapr.NewLogger(zap.L()).WithValues("eventType", eventType, "deliverID", deliveryID)  // want `missing traceId in logging keys`
	log = log.WithValues("eventType", "hello")
	log.Info("Tracing")
	return nil
}