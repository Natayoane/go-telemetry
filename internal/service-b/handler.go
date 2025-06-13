package serviceb

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/Natayoane/go-telemetry/configs"
	"github.com/Natayoane/go-telemetry/shared/models"
	"github.com/Natayoane/go-telemetry/shared/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Handler struct {
	config configs.Config
}

func NewHandler(config configs.Config) *Handler {
	return &Handler{config: config}
}

func (h *Handler) HandleTemperatureRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := telemetry.GetTracer("service-b")

	ctx, span := tracer.Start(ctx, "handle_temperature_request")
	defer span.End()

	if r.Method != http.MethodPost {
		span.SetStatus(codes.Error, "Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CEPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "Failed to decode request")
		span.RecordError(err)
		writeErrorResponse(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	span.SetAttributes(attribute.String("cep", req.CEP))

	_, validationSpan := tracer.Start(ctx, "validate_cep")
	valid := isValidCEP(req.CEP)
	if valid {
		validationSpan.SetStatus(codes.Ok, "CEP validation successful")
	} else {
		validationSpan.SetStatus(codes.Error, "Invalid CEP format")
	}
	validationSpan.End()

	if !valid {
		span.SetStatus(codes.Error, "Invalid CEP")
		writeErrorResponse(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	viaCepData, err := h.getViaCepData(ctx, req.CEP)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to get CEP data")
		span.RecordError(err)
		log.Printf("Error getting CEP data: %v", err)
		writeErrorResponse(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	if viaCepData.Erro || viaCepData.Localidade == "" {
		span.SetStatus(codes.Error, "CEP not found")
		span.SetAttributes(attribute.Bool("cep.found", false))
		writeErrorResponse(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	span.SetAttributes(
		attribute.String("city", viaCepData.Localidade),
		attribute.String("state", viaCepData.Uf),
		attribute.Bool("cep.found", true),
	)

	geoData, err := h.getCoordinates(ctx, viaCepData.Localidade, viaCepData.Uf)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to get coordinates")
		span.RecordError(err)
		log.Printf("Error getting coordinates: %v", err)
		writeErrorResponse(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	span.SetAttributes(
		attribute.Float64("geo.latitude", geoData.Lat),
		attribute.Float64("geo.longitude", geoData.Lon),
	)

	weatherData, err := h.getWeatherData(ctx, geoData.Lat, geoData.Lon)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to get weather data")
		span.RecordError(err)
		log.Printf("Error getting weather data: %v", err)
		http.Error(w, "Error fetching weather data", http.StatusInternalServerError)
		return
	}

	temperature := h.parseTemperatureResponse(weatherData)

	span.SetAttributes(
		attribute.Float64("temperature.celsius", temperature.Celsius),
		attribute.Float64("temperature.fahrenheit", temperature.Fahrenheit),
		attribute.Float64("temperature.kelvin", temperature.Kelvin),
	)

	response := models.TemperatureResponse{
		City:  viaCepData.Localidade,
		TempC: temperature.Celsius,
		TempF: temperature.Fahrenheit,
		TempK: temperature.Kelvin,
	}

	span.SetStatus(codes.Ok, "Temperature request completed successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func isValidCEP(cep string) bool {
	if len(cep) != 8 {
		return false
	}

	matched, _ := regexp.MatchString(`^[0-9]{8}$`, cep)
	return matched
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{Message: message}
	json.NewEncoder(w).Encode(errorResp)
}
