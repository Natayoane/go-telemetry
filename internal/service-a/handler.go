package servicea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/Natayoane/go-telemetry/configs"
	"github.com/Natayoane/go-telemetry/shared/models"
	"github.com/Natayoane/go-telemetry/shared/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Handler struct {
	config configs.Config
}

func NewHandler(config configs.Config) *Handler {
	return &Handler{config: config}
}

func (h *Handler) HandleCEPRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := telemetry.GetTracer("service-a")

	// Create span for the entire request
	ctx, span := tracer.Start(ctx, "handle_cep_request")
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

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := h.forwardToServiceB(ctx, req.CEP)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to forward to Service B")
		span.RecordError(err)
		log.Printf("Error forwarding to Service B: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("service_b.status_code", response.StatusCode),
		attribute.String("service_b.response_size", fmt.Sprintf("%d", len(response.Body))),
	)

	if response.StatusCode >= 400 {
		span.SetStatus(codes.Error, "Service B returned error")
	} else {
		span.SetStatus(codes.Ok, "Request completed successfully")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	w.Write([]byte(response.Body))
}

func isValidCEP(cep string) bool {
	if len(cep) != 8 {
		return false
	}

	matched, _ := regexp.MatchString(`^[0-9]{8}$`, cep)
	return matched
}

func (h *Handler) forwardToServiceB(ctx context.Context, cep string) (*ServiceBResponse, error) {
	tracer := telemetry.GetTracer("service-a")

	ctx, span := tracer.Start(ctx, "forward_to_service_b")
	defer span.End()

	url := fmt.Sprintf("%s/temperature", h.config.ServiceBURL)
	span.SetAttributes(
		attribute.String("service_b.url", url),
		attribute.String("cep", cep),
	)

	reqBody := models.CEPRequest{CEP: cep}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to marshal request")
		span.RecordError(err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		span.SetStatus(codes.Error, "Failed to create request")
		span.RecordError(err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "HTTP request failed")
		span.RecordError(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to read response")
		span.RecordError(err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("http.method", "POST"),
		attribute.Int("response.size", len(body)),
	)

	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, "Service B returned error status")
	} else {
		span.SetStatus(codes.Ok, "Service B request successful")
	}

	return &ServiceBResponse{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}, nil
}

type ServiceBResponse struct {
	StatusCode int
	Body       string
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{Message: message}
	json.NewEncoder(w).Encode(errorResp)
}
