package oauth2

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrStateNotFound         = errors.New("invalid state: not found in stored states")
	ErrEncryptionFailed      = errors.New("failed to encrypt state data")
	ErrDecryptionFailed      = errors.New("failed to decrypt state data")
	ErrSerializationFailed   = errors.New("failed to serialize state data")
	ErrDeserializationFailed = errors.New("failed to deserialize state data")
)

const stateEncryptionPrefix = "v1:"

// EncryptState serializes and encrypts the data with the state
func EncryptState[T any](key []byte, state string, data T) (string, error) {
	wrapper := struct {
		OriginalState string `json:"original_state"`
		Data          T      `json:"data"`
	}{
		OriginalState: state,
		Data:          data,
	}

	jsonData, err := json.Marshal(wrapper)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSerializationFailed, err)
	}

	// Encrypt using AES-GCM for confidentiality and integrity.
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nil, nonce, jsonData, nil)
	encryptedData := append(nonce, ciphertext...)
	return stateEncryptionPrefix + base64.URLEncoding.EncodeToString(encryptedData), nil
}

// DecryptState decrypts and deserializes the encrypted state
func DecryptState[T any](key []byte, state string) (string, T, error) {
	var empty T

	isV1 := strings.HasPrefix(state, stateEncryptionPrefix)
	payload := state
	if isV1 {
		payload = strings.TrimPrefix(state, stateEncryptionPrefix)
	}

	// base64 decode
	encryptedData, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", empty, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	if isV1 {
		block, err := aes.NewCipher(key)
		if err != nil {
			return "", empty, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", empty, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
		}

		nonceSize := gcm.NonceSize()
		if len(encryptedData) < nonceSize {
			return "", empty, fmt.Errorf("%w: encrypted data too short", ErrDecryptionFailed)
		}

		nonce := encryptedData[:nonceSize]
		ciphertext := encryptedData[nonceSize:]

		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return "", empty, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
		}

		var wrapper struct {
			OriginalState string `json:"original_state"`
			Data          T      `json:"data"`
		}
		if err := json.Unmarshal(plaintext, &wrapper); err != nil {
			return "", empty, fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
		}

		return wrapper.OriginalState, wrapper.Data, nil
	}

	// Legacy CBC fallback (unauthenticated).
	// need at leas IV + one block of data
	if len(encryptedData) < aes.BlockSize*2 {
		return "", empty, fmt.Errorf("%w: encrypted data too short", ErrDecryptionFailed)
	}

	// extract IV and chipertext
	iv := encryptedData[:aes.BlockSize]
	ciphertext := encryptedData[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", empty, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	unpaddedData, err := pkcs7Unpad(plaintext)
	if err != nil {
		return "", empty, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	var wrapper struct {
		OriginalState string `json:"original_state"`
		Data          T      `json:"data"`
	}
	if err := json.Unmarshal(unpaddedData, &wrapper); err != nil {
		return "", empty, fmt.Errorf("%w: %v", ErrDeserializationFailed, err)
	}

	return wrapper.OriginalState, wrapper.Data, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[len(data)-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, fmt.Errorf("invalid padding size")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return data[:len(data)-padding], nil
}
