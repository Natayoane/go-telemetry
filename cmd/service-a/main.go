package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Natayoane/go-telemetry/configs"
	servicea "github.com/Natayoane/go-telemetry/internal/service-a"
	"github.com/Natayoane/go-telemetry/shared/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	shutdown := telemetry.InitTelemetry("service-a", "v1.0.0")
	defer shutdown()

	config := configs.LoadConfig()
	handler := servicea.NewHandler(config)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.HandleCEPRequest)

	instrumentedHandler := otelhttp.NewHandler(mux, "service-a")

	port := "8080"
	fmt.Printf("Service A starting on port %s...\n", port)
	fmt.Printf("Service B URL configured as: %s\n", config.ServiceBURL)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(":"+port, instrumentedHandler); err != nil {
			log.Fatal(err)
		}
	}()

	<-c
	fmt.Println("Service A shutting down...")
}
