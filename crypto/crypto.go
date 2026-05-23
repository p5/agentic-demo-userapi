// Package crypto provides cryptographic signature functions using HMAC-SHA256.
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrInvalidSignature is returned when a signature verification fails due to mismatch.
var ErrInvalidSignature = errors.New("crypto: invalid signature")

// SignPayload computes HMAC-SHA256 of payload using secret and returns the hex-encoded signature string.
//
// The function creates a new HMAC instance with SHA256 as the hash function,
// writes the payload to it, and returns the resulting signature as a lowercase
// hexadecimal string.
//
// Example:
//
//	secret := []byte("my-secret-key")
//	payload := []byte("message to sign")
//	signature := SignPayload(secret, payload)
func SignPayload(secret []byte, payload []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write(payload)
	signature := h.Sum(nil)
	return hex.EncodeToString(signature)
}

// VerifySignature checks that the provided hex-encoded signature matches the HMAC-SHA256 of payload with secret.
//
// Returns nil on success, ErrInvalidSignature if the signature does not match,
// or an error if the signature is not valid hexadecimal encoding.
//
// The function uses constant-time comparison (hmac.Equal) to prevent timing attacks.
//
// Example:
//
//	secret := []byte("my-secret-key")
//	payload := []byte("message to verify")
//	signature := "a1b2c3d4..." // hex-encoded signature
//	err := VerifySignature(secret, payload, signature)
//	if err != nil {
//	    // handle error
//	}
func VerifySignature(secret []byte, payload []byte, signature string) error {
	// Decode the hex-encoded signature
	providedSig, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("crypto: invalid hex signature: %w", err)
	}

	// Compute the expected signature
	h := hmac.New(sha256.New, secret)
	h.Write(payload)
	expectedSig := h.Sum(nil)

	// Constant-time comparison to prevent timing attacks
	if !hmac.Equal(providedSig, expectedSig) {
		return ErrInvalidSignature
	}

	return nil
}
