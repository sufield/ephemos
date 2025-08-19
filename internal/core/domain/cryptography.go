// Package domain contains cryptographic validation focused on ECDSA keys used in SPIFFE.
package domain

import (
	"crypto"
	"crypto/ecdsa"
	"fmt"
)

// SupportsKeyType checks if a key is a supported ECDSA key type.
// For MVP, we only support ECDSA keys as they are the standard for SPIFFE.
func SupportsKeyType(key interface{}) bool {
	switch key.(type) {
	case *ecdsa.PublicKey, *ecdsa.PrivateKey:
		return true
	default:
		return false
	}
}

// ValidateKeyPairMatching validates that a private key matches a certificate's public key.
// This replaces mechanical comparison operations with domain intent validation.
func ValidateKeyPairMatching(certPublicKey interface{}, privateKeyPublic interface{}) error {
	// Express intent: verify this is a supported ECDSA key type
	if !SupportsKeyType(certPublicKey) {
		return fmt.Errorf("certificate public key must be ECDSA for SPIFFE - unsupported key type: %T", certPublicKey)
	}

	if !SupportsKeyType(privateKeyPublic) {
		return fmt.Errorf("private key must be ECDSA for SPIFFE - unsupported key type: %T", privateKeyPublic)
	}

	// Express intent: verify the keys match cryptographically
	if !KeysMatch(certPublicKey, privateKeyPublic) {
		return fmt.Errorf("ECDSA key pair does not match")
	}

	return nil
}

// KeysMatch determines if two ECDSA cryptographic keys represent the same key pair.
// This replaces mechanical equal operations with domain intent.
func KeysMatch(certPublicKey, privateKeyPublic interface{}) bool {
	// Both keys must be ECDSA for SPIFFE
	ecdsaCertKey, ok1 := certPublicKey.(*ecdsa.PublicKey)
	ecdsaPrivKey, ok2 := privateKeyPublic.(*ecdsa.PublicKey)
	
	if !ok1 || !ok2 {
		return false
	}
	
	// Compare ECDSA public key components: curve and coordinates (X, Y)
	return ecdsaCertKey.Curve == ecdsaPrivKey.Curve &&
		   ecdsaCertKey.X.Cmp(ecdsaPrivKey.X) == 0 &&
		   ecdsaCertKey.Y.Cmp(ecdsaPrivKey.Y) == 0
}

// ValidateSignerKeyType validates that a crypto.Signer is ECDSA.
// This replaces mechanical type switches with domain intent validation.
func ValidateSignerKeyType(signer crypto.Signer) error {
	if signer == nil {
		return fmt.Errorf("signer cannot be nil")
	}

	if !SupportsKeyType(signer) {
		return fmt.Errorf("signer must be ECDSA for SPIFFE - unsupported signer type: %T", signer)
	}

	return nil
}

// ExtractPublicKeyFromSigner safely extracts the public key from an ECDSA crypto.Signer.
// This wraps the mechanical Public() call with domain intent and validation.
func ExtractPublicKeyFromSigner(signer crypto.Signer) (interface{}, error) {
	if err := ValidateSignerKeyType(signer); err != nil {
		return nil, fmt.Errorf("invalid signer: %w", err)
	}

	publicKey := signer.Public()
	if publicKey == nil {
		return nil, fmt.Errorf("signer returned nil public key")
	}

	// Verify the extracted public key is ECDSA
	if !SupportsKeyType(publicKey) {
		return nil, fmt.Errorf("signer returned non-ECDSA public key: %T", publicKey)
	}

	return publicKey, nil
}

// ValidatedKeyPair represents an ECDSA key pair that has been validated to match.
// This value object guarantees that the public and private keys are cryptographically paired,
// eliminating the need for repeated validation checks in consuming code.
type ValidatedKeyPair struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

// NewValidatedKeyPair creates a new ValidatedKeyPair after validating that the ECDSA keys match.
// This constructor ensures that only valid ECDSA key pairs can be created.
func NewValidatedKeyPair(publicKey interface{}, privateKey crypto.Signer) (*ValidatedKeyPair, error) {
	// Validate that we have ECDSA keys
	ecdsaPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key must be ECDSA, got: %T", publicKey)
	}

	ecdsaPrivateKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key must be ECDSA, got: %T", privateKey)
	}

	// Extract public key from private key for comparison
	privateKeyPublic := ecdsaPrivateKey.Public()

	// Validate that the key pair matches
	if err := ValidateKeyPairMatching(ecdsaPublicKey, privateKeyPublic); err != nil {
		return nil, fmt.Errorf("ECDSA key pair validation failed: %w", err)
	}

	return &ValidatedKeyPair{
		publicKey:  ecdsaPublicKey,
		privateKey: ecdsaPrivateKey,
	}, nil
}

// PublicKey returns the validated ECDSA public key.
func (vkp *ValidatedKeyPair) PublicKey() *ecdsa.PublicKey {
	return vkp.publicKey
}

// PrivateKey returns the validated ECDSA private key.
func (vkp *ValidatedKeyPair) PrivateKey() *ecdsa.PrivateKey {
	return vkp.privateKey
}

// ValidateAgainstCertificate verifies that this ECDSA key pair matches the given certificate.
func (vkp *ValidatedKeyPair) ValidateAgainstCertificate(cert interface{ PublicKey() interface{} }) error {
	certPublicKey := cert.PublicKey()
	if !KeysMatch(certPublicKey, vkp.publicKey) {
		return fmt.Errorf("validated ECDSA key pair does not match certificate public key")
	}
	return nil
}