package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealthEndpoint(t *testing.T) {
	app := NewServer()

	request, err := http.NewRequest(http.MethodGet, healthPath, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != fiber.StatusOK {
		t.Errorf(
			"status code = %d, want %d",
			response.StatusCode,
			fiber.StatusOK,
		)
	}

	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Status != healthStatus {
		t.Errorf("status = %q, want %q", body.Status, healthStatus)
	}

	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q, want JSON", contentType)
	}
}
