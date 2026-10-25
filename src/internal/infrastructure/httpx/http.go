package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// ErrorResponse is the stable JSON shape used for API failures.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains a machine-readable code and safe client message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// DecodeJSON decodes one JSON object and rejects unknown fields or trailing values.
func DecodeJSON(ctx *fiber.Ctx, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(ctx.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON request: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON value")
	}
	return nil
}

// WriteError writes a consistent API error response.
func WriteError(ctx *fiber.Ctx, status int, code, message string) error {
	return ctx.Status(status).JSON(ErrorResponse{
		Error: ErrorDetail{Code: code, Message: message},
	})
}

// WriteInternalError logs an unexpected error and hides its details from clients.
func WriteInternalError(ctx *fiber.Ctx, err error) error {
	slog.Error("request failed", "error", err)
	return WriteError(ctx, fiber.StatusInternalServerError, "internal_error", "An unexpected error occurred")
}
