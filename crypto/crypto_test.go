package crypto

import (
	"errors"
	"testing"
)

// TestSignPayload tests that SignPayload produces the expected hex-encoded HMAC-SHA256 signature.
func TestSignPayload(t *testing.T) {
	secret := []byte("my-secret-key")
	payload := []byte("hello world")
	
	// Expected HMAC-SHA256 signature computed independently
	// echo -n "hello world" | openssl dgst -sha256 -hmac "my-secret-key"
	expected := "734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a"
	
	signature := SignPayload(secret, payload)
	
	if signature != expected {
		t.Errorf("SignPayload() = %q, want %q", signature, expected)
	}
	
	// Verify the signature is valid hex and has expected length (64 chars for SHA256)
	if len(signature) != 64 {
		t.Errorf("SignPayload() returned signature of length %d, want 64", len(signature))
	}
}

// TestVerifySignature_Valid tests that a valid signature is successfully verified.
func TestVerifySignature_Valid(t *testing.T) {
	secret := []byte("test-secret")
	payload := []byte("test payload")
	
	// Sign the payload
	signature := SignPayload(secret, payload)
	
	// Verify the signature
	err := VerifySignature(secret, payload, signature)
	if err != nil {
		t.Errorf("VerifySignature() returned error for valid signature: %v", err)
	}
}

// TestVerifySignature_Invalid tests that an invalid signature returns ErrInvalidSignature.
func TestVerifySignature_Invalid(t *testing.T) {
	secret := []byte("test-secret")
	payload := []byte("test payload")
	
	// Use a wrong signature (valid hex, but incorrect value)
	wrongSignature := "0000000000000000000000000000000000000000000000000000000000000000"
	
	err := VerifySignature(secret, payload, wrongSignature)
	if err == nil {
		t.Error("VerifySignature() returned nil for invalid signature, want error")
	}
	
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("VerifySignature() error = %v, want ErrInvalidSignature", err)
	}
}

// TestVerifySignature_BadHex tests that a non-hex signature string returns an error (not ErrInvalidSignature).
func TestVerifySignature_BadHex(t *testing.T) {
	secret := []byte("test-secret")
	payload := []byte("test payload")
	
	// Use invalid hex characters
	badHexSignature := "not-valid-hex-xyz"
	
	err := VerifySignature(secret, payload, badHexSignature)
	if err == nil {
		t.Error("VerifySignature() returned nil for bad hex signature, want error")
	}
	
	// Should NOT be ErrInvalidSignature (should be hex decoding error)
	if errors.Is(err, ErrInvalidSignature) {
		t.Errorf("VerifySignature() returned ErrInvalidSignature for bad hex, want hex decoding error")
	}
}

// TestVerifySignature_EmptyPayload tests that empty payload with valid signature works correctly.
func TestVerifySignature_EmptyPayload(t *testing.T) {
	secret := []byte("test-secret")
	payload := []byte("")
	
	// Sign the empty payload
	signature := SignPayload(secret, payload)
	
	// Verify the signature
	err := VerifySignature(secret, payload, signature)
	if err != nil {
		t.Errorf("VerifySignature() returned error for valid signature with empty payload: %v", err)
	}
	
	// Also verify that the signature is not empty
	if signature == "" {
		t.Error("SignPayload() returned empty signature for empty payload")
	}
}

// TestSignPayload_Consistency tests that signing the same data produces consistent results.
func TestSignPayload_Consistency(t *testing.T) {
	secret := []byte("consistent-secret")
	payload := []byte("consistent payload")
	
	sig1 := SignPayload(secret, payload)
	sig2 := SignPayload(secret, payload)
	
	if sig1 != sig2 {
		t.Errorf("SignPayload() inconsistent: first=%q, second=%q", sig1, sig2)
	}
}

// TestSignPayload_DifferentSecrets tests that different secrets produce different signatures.
func TestSignPayload_DifferentSecrets(t *testing.T) {
	secret1 := []byte("secret-one")
	secret2 := []byte("secret-two")
	payload := []byte("same payload")
	
	sig1 := SignPayload(secret1, payload)
	sig2 := SignPayload(secret2, payload)
	
	if sig1 == sig2 {
		t.Error("SignPayload() produced same signature for different secrets")
	}
}

// TestVerifySignature_WrongSecret tests that verification fails with wrong secret.
func TestVerifySignature_WrongSecret(t *testing.T) {
	secret1 := []byte("original-secret")
	secret2 := []byte("wrong-secret")
	payload := []byte("test payload")
	
	// Sign with secret1
	signature := SignPayload(secret1, payload)
	
	// Try to verify with secret2
	err := VerifySignature(secret2, payload, signature)
	if err == nil {
		t.Error("VerifySignature() returned nil when using wrong secret, want error")
	}
	
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("VerifySignature() error = %v, want ErrInvalidSignature", err)
	}
}

// TestVerifySignature_ModifiedPayload tests that verification fails if payload is modified.
func TestVerifySignature_ModifiedPayload(t *testing.T) {
	secret := []byte("test-secret")
	originalPayload := []byte("original payload")
	modifiedPayload := []byte("modified payload")
	
	// Sign original payload
	signature := SignPayload(secret, originalPayload)
	
	// Try to verify with modified payload
	err := VerifySignature(secret, modifiedPayload, signature)
	if err == nil {
		t.Error("VerifySignature() returned nil for modified payload, want error")
	}
	
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("VerifySignature() error = %v, want ErrInvalidSignature", err)
	}
}
