package observability

import (
	"os"
	"strconv"
	"strings"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func samplerFromEnv() sdktrace.Sampler {
	switch strings.ToLower(os.Getenv("OTEL_TRACES_SAMPLER")) {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratioFromEnv())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratioFromEnv()))
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

func ratioFromEnv() float64 {
	raw := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	if raw == "" {
		return 1
	}

	ratio, err := strconv.ParseFloat(raw, 64)
	if err != nil || ratio < 0 || ratio > 1 {
		return 1
	}

	return ratio
}
