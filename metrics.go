package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials"
)

var (
	mTCPBytesSent     metric.Int64Counter
	mTCPBytesReceived metric.Int64Counter
	mTCPConnections   metric.Int64Counter
	mUDPBytesSent     metric.Int64Counter
	mUDPBytesReceived metric.Int64Counter
)

// initMetrics sets up the OTel metrics provider pointing at a Grafana Cloud OTLP endpoint.
// nodeName becomes service.instance.id in Grafana.
// endpoint is the gRPC host:port (e.g. "tempo-prod-06-prod-eu-west-0.grafana.net:443").
// token is the base64-encoded "instanceID:apitoken" for HTTP Basic auth.
func initMetrics(ctx context.Context, nodeName, endpoint, token string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("shadowsocks"),
			semconv.ServiceInstanceID(nodeName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"Authorization": "Basic " + token,
		}),
		otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	otel.SetMeterProvider(provider)

	meter := provider.Meter("github.com/shadowsocks/go-shadowsocks2")

	if mTCPBytesSent, err = meter.Int64Counter("ss.tcp.bytes_sent",
		metric.WithDescription("Total TCP bytes sent to clients (downstream)"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("create metric ss.tcp.bytes_sent: %w", err)
	}

	if mTCPBytesReceived, err = meter.Int64Counter("ss.tcp.bytes_received",
		metric.WithDescription("Total TCP bytes received from clients (upstream)"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("create metric ss.tcp.bytes_received: %w", err)
	}

	if mTCPConnections, err = meter.Int64Counter("ss.tcp.connections_total",
		metric.WithDescription("Total TCP connections handled"),
	); err != nil {
		return nil, fmt.Errorf("create metric ss.tcp.connections_total: %w", err)
	}

	if mUDPBytesSent, err = meter.Int64Counter("ss.udp.bytes_sent",
		metric.WithDescription("Total UDP bytes sent to clients (downstream)"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("create metric ss.udp.bytes_sent: %w", err)
	}

	if mUDPBytesReceived, err = meter.Int64Counter("ss.udp.bytes_received",
		metric.WithDescription("Total UDP bytes received from clients (upstream)"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("create metric ss.udp.bytes_received: %w", err)
	}

	return provider.Shutdown, nil
}

func recordTCPTraffic(downstream, upstream int64) {
	if mTCPBytesSent == nil {
		return
	}
	ctx := context.Background()
	mTCPBytesSent.Add(ctx, downstream)
	mTCPBytesReceived.Add(ctx, upstream)
	mTCPConnections.Add(ctx, 1)
}

func recordUDPSent(n int64) {
	if mUDPBytesSent == nil {
		return
	}
	mUDPBytesSent.Add(context.Background(), n)
}

func recordUDPReceived(n int64) {
	if mUDPBytesReceived == nil {
		return
	}
	mUDPBytesReceived.Add(context.Background(), n)
}
