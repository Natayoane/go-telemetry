package serviceb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	httputils "github.com/Natayoane/go-telemetry/shared/http-utils"
	"github.com/Natayoane/go-telemetry/shared/models"
	"github.com/Natayoane/go-telemetry/shared/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Temperature struct {
	Celsius    float64
	Fahrenheit float64
	Kelvin     float64
}

func (h *Handler) getViaCepData(ctx context.Context, cep string) (models.ViaCepData, error) {
	tracer := telemetry.GetTracer("service-b")

	ctx, span := tracer.Start(ctx, "fetch_viacep_data")
	defer span.End()

	var result models.ViaCepData
	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)

	span.SetAttributes(
		attribute.String("cep", cep),
		attribute.String("api.name", "viacep"),
		attribute.String("api.url", url),
	)

	log.Printf("CEP API URL: %s", url)

	response, err := httputils.FetchAPI(ctx, url)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to fetch CEP data")
		span.RecordError(err)
		log.Printf("Error fetching CEP data: %v", err)
		return models.ViaCepData{}, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", response.StatusCode),
		attribute.Int("response.size", len(response.Body)),
	)

	log.Printf("CEP API Response: %s", response.Body)

	if err := json.Unmarshal([]byte(response.Body), &result); err != nil {
		span.SetStatus(codes.Error, "Failed to parse CEP response")
		span.RecordError(err)
		log.Printf("Error unmarshaling CEP data: %v", err)
		return models.ViaCepData{}, err
	}

	if result.Erro {
		span.SetStatus(codes.Error, "CEP not found")
		span.SetAttributes(attribute.Bool("cep.found", false))
	} else {
		span.SetStatus(codes.Ok, "CEP data fetched successfully")
		span.SetAttributes(
			attribute.Bool("cep.found", true),
			attribute.String("city", result.Localidade),
			attribute.String("state", result.Uf),
		)
	}

	log.Printf("CEP Data: %+v", result)
	return result, nil
}

func (h *Handler) getCoordinates(ctx context.Context, city string, state string) (models.GeoData, error) {
	tracer := telemetry.GetTracer("service-b")

	ctx, span := tracer.Start(ctx, "fetch_coordinates")
	defer span.End()

	var result []models.GeoData

	encodedCity := url.QueryEscape(city)
	encodedState := url.QueryEscape(state)

	apiURL := fmt.Sprintf("https://api.openweathermap.org/geo/1.0/direct?q=%s,%s,BR&limit=1&appid=%s", encodedCity, encodedState, h.config.OpenWeatherMapAPIKey)

	span.SetAttributes(
		attribute.String("city", city),
		attribute.String("state", state),
		attribute.String("api.name", "openweathermap_geocoding"),
		attribute.String("api.url", apiURL), // Note: this will include the API key, you might want to redact it
	)

	log.Printf("Geocoding API URL: %s", apiURL)

	response, err := httputils.FetchAPI(ctx, apiURL)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to fetch coordinates")
		span.RecordError(err)
		log.Printf("Error fetching coordinates: %v", err)
		return models.GeoData{}, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", response.StatusCode),
		attribute.Int("response.size", len(response.Body)),
	)

	log.Printf("Geocoding API Response: %s", response.Body)

	if err := json.Unmarshal([]byte(response.Body), &result); err != nil {
		span.SetStatus(codes.Error, "Failed to parse geocoding response")
		span.RecordError(err)
		log.Printf("Error unmarshaling coordinates: %v", err)
		return models.GeoData{}, err
	}

	if len(result) == 0 {
		span.SetStatus(codes.Error, "No coordinates found for city")
		span.SetAttributes(attribute.Bool("coordinates.found", false))
		return models.GeoData{}, fmt.Errorf("city not found")
	}

	span.SetAttributes(
		attribute.Bool("coordinates.found", true),
		attribute.Float64("latitude", result[0].Lat),
		attribute.Float64("longitude", result[0].Lon),
	)
	span.SetStatus(codes.Ok, "Coordinates fetched successfully")

	return result[0], nil
}

func (h *Handler) getWeatherData(ctx context.Context, lat, lon float64) (models.WeatherData, error) {
	tracer := telemetry.GetTracer("service-b")

	ctx, span := tracer.Start(ctx, "fetch_weather_data")
	defer span.End()

	var result models.WeatherData
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%.6f&lon=%.6f&appid=%s", lat, lon, h.config.OpenWeatherMapAPIKey)

	span.SetAttributes(
		attribute.Float64("latitude", lat),
		attribute.Float64("longitude", lon),
		attribute.String("api.name", "openweathermap_weather"),
		attribute.String("api.url", url), // Note: this will include the API key, you might want to redact it
	)

	log.Printf("Weather API URL: %s", url)

	response, err := httputils.FetchAPI(ctx, url)
	if err != nil {
		span.SetStatus(codes.Error, "Failed to fetch weather data")
		span.RecordError(err)
		log.Printf("Error fetching weather data: %v", err)
		return models.WeatherData{}, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", response.StatusCode),
		attribute.Int("response.size", len(response.Body)),
	)

	log.Printf("Weather API Response: %s", response.Body)

	if err := json.Unmarshal([]byte(response.Body), &result); err != nil {
		span.SetStatus(codes.Error, "Failed to parse weather response")
		span.RecordError(err)
		log.Printf("Error unmarshaling weather data: %v", err)
		return models.WeatherData{}, err
	}

	span.SetAttributes(
		attribute.Float64("temperature.kelvin", result.Main.Temp),
		attribute.Float64("temperature.feels_like", result.Main.FeelsLike),
		attribute.Int("pressure", result.Main.Pressure),
		attribute.Int("humidity", result.Main.Humidity),
	)
	span.SetStatus(codes.Ok, "Weather data fetched successfully")

	return result, nil
}

func (h *Handler) parseTemperatureResponse(weather models.WeatherData) Temperature {
	log.Printf("Raw weather data: %+v", weather)

	kelvin := weather.Main.Temp
	log.Printf("Temperature in Kelvin: %f", kelvin)

	celsius := kelvin - 273.15
	log.Printf("Temperature in Celsius: %f", celsius)

	fahrenheit := celsius*1.8 + 32
	log.Printf("Temperature in Fahrenheit: %f", fahrenheit)

	temp := Temperature{
		Celsius:    celsius,
		Fahrenheit: fahrenheit,
		Kelvin:     kelvin,
	}

	log.Printf("Final temperature object: %+v", temp)
	return temp
}
