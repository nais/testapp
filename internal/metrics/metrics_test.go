package metrics_test

import (
	"testing"

	"github.com/nais/testapp/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMetricsRegistration(t *testing.T) {
	// Test that metrics are properly registered
	metricsList := []prometheus.Collector{
		metrics.LeadTime,
		metrics.TimeSinceDeploy,
		metrics.DeployTimestamp,
		metrics.StartTimestamp,
	}

	for _, metric := range metricsList {
		if metric == nil {
			t.Error("metric should not be nil")
		}
	}
}

func TestMetricsHandler(t *testing.T) {
	handler := metrics.Handler()
	if handler == nil {
		t.Fatal("metrics handler should not be nil")
	}
}

func TestGaugeMetrics(t *testing.T) {
	tests := []struct {
		name   string
		metric prometheus.Gauge
		value  float64
	}{
		{"LeadTime", metrics.LeadTime, 123.45},
		{"TimeSinceDeploy", metrics.TimeSinceDeploy, 678.90},
		{"DeployTimestamp", metrics.DeployTimestamp, 1234567890.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.metric.Set(tt.value)

			var m dto.Metric
			if err := tt.metric.Write(&m); err != nil {
				t.Fatalf("failed to write metric: %v", err)
			}

			if m.Gauge == nil {
				t.Fatal("expected gauge metric")
			}

			if got := m.Gauge.GetValue(); got != tt.value {
				t.Errorf("expected %f, got %f", tt.value, got)
			}
		})
	}
}

func TestStartTimestamp(t *testing.T) {
	// Test that StartTimestamp can be set to current time
	metrics.StartTimestamp.SetToCurrentTime()

	var m dto.Metric
	if err := metrics.StartTimestamp.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.Gauge == nil {
		t.Fatal("expected gauge metric")
	}

	// Should be a recent timestamp (positive value)
	if got := m.Gauge.GetValue(); got <= 0 {
		t.Errorf("expected positive timestamp, got %f", got)
	}
}
