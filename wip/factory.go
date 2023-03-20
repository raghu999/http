package httplogreceiver

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr        = "httplogreceiver"
	stability      = component.StabilityLevelAlpha
	defaultEndpoint = ":8888"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, stability))
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint: defaultEndpoint,
	}
}

func createLogsReceiver(
	ctx context.Context,
	params receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	if nextConsumer == nil {
		return nil, errors.New("logsNextConsumer is nil")
	}

	config := cfg.(*Config)

	return NewReceiver(params, config, nextConsumer)
}
