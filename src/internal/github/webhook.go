package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidSignature = errors.New("invalid GitHub webhook signature")
	ErrMissingSecret    = errors.New("GitHub webhook secret is not configured")
	ErrMalformedPayload = errors.New("malformed GitHub webhook payload")
)

// PushEvent represents the subset of a GitHub push webhook payload used for deployment validation.
type PushEvent struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// VerifySignature validates the SHA256 HMAC using the request body in its original byte form.
func VerifySignature(secret, providedSignature string, body []byte) error {
	if strings.TrimSpace(secret) == "" {
		return ErrMissingSecret
	}
	if strings.TrimSpace(providedSignature) == "" {
		return ErrInvalidSignature
	}

	expected := hmac.New(sha256.New, []byte(secret))
	if _, err := expected.Write(body); err != nil {
		return fmt.Errorf("compute HMAC: %w", err)
	}
	summed := expected.Sum(nil)
	expectedHex := hex.EncodeToString(summed)
	actual := strings.TrimSpace(providedSignature)
	actual = strings.TrimPrefix(actual, "sha256=")
	if len(actual) != len(expectedHex) {
		return ErrInvalidSignature
	}
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHex)) != 1 {
		return ErrInvalidSignature
	}
	return nil
}

// ParsePushEvent decodes a GitHub push payload and rejects malformed data.
func ParsePushEvent(body []byte) (PushEvent, error) {
	var event PushEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return PushEvent{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
	}
	if strings.TrimSpace(event.Ref) == "" || strings.TrimSpace(event.After) == "" {
		return PushEvent{}, ErrMalformedPayload
	}
	return event, nil
}

// IsDeployableCommitSHA ensures the value is a full Git SHA, not a branch deletion marker.
func IsDeployableCommitSHA(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || strings.TrimLeft(value, "0") == "" {
		return false
	}
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') &&
			(character < 'A' || character > 'F') {
			return false
		}
	}
	return true
}

// MatchesConfiguredBranch accepts either a bare branch name or a full refs/heads/... reference.
func MatchesConfiguredBranch(configuredBranch, ref string) bool {
	configuredBranch = strings.TrimSpace(configuredBranch)
	ref = strings.TrimSpace(ref)
	if configuredBranch == "" || ref == "" {
		return false
	}
	if strings.HasPrefix(configuredBranch, "refs/heads/") {
		return ref == configuredBranch
	}
	if strings.HasPrefix(ref, "refs/heads/") {
		return ref == "refs/heads/"+configuredBranch
	}
	return ref == configuredBranch
}
