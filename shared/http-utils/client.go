package httputils

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
)

type APIResponse struct {
	Body       string
	StatusCode int
}

func FetchAPI(ctx context.Context, url string) (APIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return APIResponse{}, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return APIResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return APIResponse{}, err
	}

	log.Printf("API Response Status: %d", res.StatusCode)
	log.Printf("API Response Body: %s", string(body))

	if res.StatusCode != http.StatusOK {
		return APIResponse{}, fmt.Errorf("API returned status %d: %s", res.StatusCode, string(body))
	}

	return APIResponse{
		Body:       string(body),
		StatusCode: res.StatusCode,
	}, nil
}
