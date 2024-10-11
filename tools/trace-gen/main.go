package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func newStdoutExporter(ctx context.Context) *stdouttrace.Exporter /* (someExporter.Exporter, error) */ {

	exp, err := stdouttrace.New()

	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	return exp
}

func newOltpGrpcExporter(ctx context.Context) *otlptrace.Exporter {

	endpointOpt := otlptracegrpc.WithEndpoint("localhost:9987")
	insecureOpt := otlptracegrpc.WithInsecure()

	exp, err := otlptracegrpc.New(ctx, endpointOpt, insecureOpt)

	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	return exp
}

func newTraceProvider(serviceName string, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	// = TraceFactory
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func randomThreadSleep(max int) {
	// 현재 시간을 시드로 사용하여 랜덤 생성기 초기화
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// 최대 max 지속시간 생성
	randomDuration := time.Duration(rand.Intn(max)+1) * time.Second

	fmt.Printf("Sleeping for %v...\n", randomDuration)
	time.Sleep(randomDuration)
	fmt.Println("Awake!")
}

func createNdepthSpan(n int, tracer trace.Tracer, ctx context.Context) {

	var spanList []trace.Span
	var span trace.Span

	for i := 0; i < n; i++ {
		// Span 이름을 형식화
		spanName := fmt.Sprintf("%v Span", i)

		// 새로운 Span 시작
		ctx, span = tracer.Start(ctx, spanName)
		randomThreadSleep(2)

		spanList = append(spanList, span)
	}

	for i := len(spanList) - 1; i >= 0; i-- {
		spanList[i].End()
	}

}

func newSpanFromProvider(tp *sdktrace.TracerProvider) {
	tracer = tp.Tracer("example.io/package/name")
}

func main() {
	ctx := context.Background()
	exp := newOltpGrpcExporter(ctx)

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := newTraceProvider("servicd-A", exp)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	tracer = tp.Tracer("example.io/package/name")

	ctx, span := tracer.Start(ctx, "hello-span")

	defer span.End()

	createNdepthSpan(5, tracer, ctx)

	fmt.Println("TraceGen Done")
}
