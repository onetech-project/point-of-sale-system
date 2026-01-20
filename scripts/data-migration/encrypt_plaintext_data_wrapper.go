package main

// EncryptPlaintextDataWrapper is the wrapper function to encrypt plaintext PII data
func EncryptPlaintextDataWrapper(config *Config) error {
	return EncryptPlaintextData(config)
}
