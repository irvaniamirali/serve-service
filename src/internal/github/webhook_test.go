package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"ref":"refs/heads/main","after":"abcdef0123456789abcdef0123456789abcdef01"}`)
	signature := "sha256=" + sign(secret, payload)
	if err := VerifySignature(secret, signature, payload); err != nil {
		t.Fatalf("VerifySignature() unexpected error: %v", err)
	}
}

func TestVerifySignatureRejectsInvalid(t *testing.T) {
	if err := VerifySignature("secret", "sha256=deadbeef", []byte("payload")); err == nil {
		t.Fatal("VerifySignature() expected invalid signature error")
	}
}

func TestMatchesConfiguredBranch(t *testing.T) {
	if !MatchesConfiguredBranch("main", "refs/heads/main") {
		t.Fatal("main branch should match refs/heads/main")
	}
	if MatchesConfiguredBranch("main", "refs/heads/main-feature") {
		t.Fatal("main should not match main-feature")
	}
}

func TestIsDeployableCommitSHA(t *testing.T) {
	if !IsDeployableCommitSHA("abcdef0123456789abcdef0123456789abcdef01") {
		t.Fatal("40-character SHA should be accepted")
	}
	if IsDeployableCommitSHA("0000000000000000000000000000000000000000") {
		t.Fatal("all-zero SHA should be rejected")
	}
}

func sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
