package manjo

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// ParsePrivateKey parses an RSA private key (PKCS#1 or PKCS#8). Besides PEM it
// accepts PEM with literal "\n" escapes and a bare base64 key body, the forms
// a key usually takes in an environment variable.
func ParsePrivateKey(value string) (*rsa.PrivateKey, error) {
	der, err := keyDER(value)
	if err != nil {
		return nil, fmt.Errorf("manjo: private key: %w", err)
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("manjo: parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("manjo: private key is not an RSA key")
	}
	return key, nil
}

// ParsePublicKey parses an RSA public key (PKIX or PKCS#1), in the same forms
// ParsePrivateKey accepts.
func ParsePublicKey(value string) (*rsa.PublicKey, error) {
	der, err := keyDER(value)
	if err != nil {
		return nil, fmt.Errorf("manjo: public key: %w", err)
	}
	if key, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("manjo: parse public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("manjo: public key is not an RSA key")
	}
	return key, nil
}

func keyDER(value string) ([]byte, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
	if value == "" {
		return nil, errors.New("empty key")
	}
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return block.Bytes, nil
	}
	der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
	if err != nil {
		return nil, errors.New("key is neither PEM nor base64")
	}
	return der, nil
}
