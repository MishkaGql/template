package db

import (
	"context"
	"regexp"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const dbInstrumentationName = "github.com/MishkaGql/template/db"

type queryStartKey struct{}

type queryInfo struct {
	operation  string
	collection string
	startedAt  time.Time
}

type collectionTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func newCollectionTracer(mp metric.MeterProvider) *collectionTracer {
	if mp == nil {
		mp = noop.NewMeterProvider()
	}

	duration, err := mp.Meter(dbInstrumentationName).Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Длительность запроса в БД"),
	)
	if err != nil {
		duration, _ = noop.NewMeterProvider().Meter(dbInstrumentationName).Float64Histogram("db.client.operation.duration")
	}

	return &collectionTracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithMeterProvider(noop.NewMeterProvider()),
		),
		duration: duration,
	}
}

func (t *collectionTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)

	collection := tableName(data.SQL)
	if collection != "" {
		trace.SpanFromContext(ctx).SetAttributes(semconv.DBCollectionName(collection))
	}

	info := queryInfo{
		operation:  otelpgx.SQLOperationName(ctx, data.SQL),
		collection: collection,
		startedAt:  time.Now(),
	}

	return context.WithValue(ctx, queryStartKey{}, info)
}

func (t *collectionTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.Tracer.TraceQueryEnd(ctx, conn, data)

	info, ok := ctx.Value(queryStartKey{}).(queryInfo)
	if !ok {
		return
	}

	attrs := []attribute.KeyValue{
		semconv.DBOperationName(info.operation),
	}

	if info.collection != "" {
		attrs = append(attrs, semconv.DBCollectionName(info.collection))
	}

	if data.Err != nil {
		attrs = append(attrs, semconv.ErrorType(data.Err))
		trace.SpanFromContext(ctx).SetAttributes(semconv.ErrorType(data.Err))
	}

	t.duration.Record(ctx, time.Since(info.startedAt).Seconds(), metric.WithAttributes(attrs...))
}

var tableNamePattern = regexp.MustCompile(`(?is)^\s*(?:insert\s+into|update|delete\s+from|from)\s+([a-z_][a-z0-9_]*)`)

func tableName(sql string) string {
	match := tableNamePattern.FindStringSubmatch(sql)
	if len(match) < 2 {
		return ""
	}

	return match[1]
}
