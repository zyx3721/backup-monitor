package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const settingsSecretPrefix = "enc:v1:"
const settingsSecretFixedKey = "backup-monitor::settings::ldap-bind-password::v1"

func DecryptSettingsSecret(raw, legacyKey string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, settingsSecretPrefix) {
		return value, nil
	}
	plain, err := decryptSettingsSecretWithKey(value, settingsSecretFixedKey)
	if err == nil {
		return plain, nil
	}
	legacy := strings.TrimSpace(legacyKey)
	if legacy != "" && legacy != settingsSecretFixedKey {
		if plain, legacyErr := decryptSettingsSecretWithKey(value, legacy); legacyErr == nil {
			return plain, nil
		}
	}
	return "", err
}

func EncryptSettingsSecret(raw string) (string, error) { return encryptSettingsSecret(raw) }

func encryptSettingsSecret(raw string) (string, error) {
	return encryptSettingsSecretWithKey(raw, settingsSecretFixedKey)
}
func encryptSettingsSecretWithKey(raw, cipherKey string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	block, err := aes.NewCipher(deriveSettingsCipherKey(cipherKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(raw), nil)
	payload := append(nonce, sealed...)
	return settingsSecretPrefix + base64.StdEncoding.EncodeToString(payload), nil
}
func decryptSettingsSecretWithKey(value, cipherKey string) (string, error) {
	encoded := strings.TrimPrefix(value, settingsSecretPrefix)
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveSettingsCipherKey(cipherKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted settings secret")
	}
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
func deriveSettingsCipherKey(raw string) []byte {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return sum[:]
}
