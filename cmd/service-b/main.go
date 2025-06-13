package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Natayoane/go-telemetry/configs"
	serviceb "github.com/Natayoane/go-telemetry/internal/service-b"
	"github.com/Natayoane/go-telemetry/shared/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	shutdown := telemetry.InitTelemetry("service-b", "v1.0.0")
	defer shutdown()

	config := configs.LoadConfig()

	if config.OpenWeatherMapAPIKey == "" {
		log.Fatal("OPENWEATHERMAP_API_KEY environment variable is required")
	}

	handler := serviceb.NewHandler(config)

	mux := http.NewServeMux()
	mux.HandleFunc("/temperature", handler.HandleTemperatureRequest)

	instrumentedHandler := otelhttp.NewHandler(mux, "service-b")

	port := "8081"
	fmt.Printf("Service B starting on port %s...\n", port)
	fmt.Printf("OpenWeatherMap API configured: %t\n", config.OpenWeatherMapAPIKey != "")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(":"+port, instrumentedHandler); err != nil {
			log.Fatal(err)
		}
	}()

	<-c
	fmt.Println("Service B shutting down...")
}
