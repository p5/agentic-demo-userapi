# crypto Package

This package provides cryptographic signature functions using HMAC-SHA256.

## Features

- **SignPayload**: Computes HMAC-SHA256 signature and returns hex-encoded string
- **VerifySignature**: Verifies hex-encoded signature using constant-time comparison
- **ErrInvalidSignature**: Package-level error for signature mismatches

## Implementation Details

- Uses only standard library packages:
  - `crypto/hmac` - HMAC implementation
  - `crypto/sha256` - SHA-256 hash function
  - `encoding/hex` - Hexadecimal encoding/decoding
  - `errors` - Error definitions
  - `fmt` - Error wrapping

- **Security**: Uses `hmac.Equal` for constant-time comparison to prevent timing attacks

## Usage Example

```go
package main

import (
    "fmt"
    "crypto"
)

func main() {
    secret := []byte("my-secret-key")
    payload := []byte("message to sign")
    
    // Sign the payload
    signature := crypto.SignPayload(secret, payload)
    fmt.Println("Signature:", signature)
    
    // Verify the signature
    err := crypto.VerifySignature(secret, payload, signature)
    if err != nil {
        fmt.Println("Verification failed:", err)
        return
    }
    fmt.Println("Signature verified successfully")
}
```

## Test Coverage

The package includes comprehensive tests:

1. **TestSignPayload**: Verifies known secret+payload produces expected hex output
2. **TestVerifySignature_Valid**: Tests signing then verifying returns nil
3. **TestVerifySignature_Invalid**: Tests wrong signature returns ErrInvalidSignature
4. **TestVerifySignature_BadHex**: Tests non-hex string returns error (not ErrInvalidSignature)
5. **TestVerifySignature_EmptyPayload**: Tests empty payload with valid signature works
6. Additional tests for consistency, different secrets, wrong secret, and modified payload

Run tests with: `go test -v`
