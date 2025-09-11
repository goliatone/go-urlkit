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
)

var (
	ErrStateNotFound         = errors.New("invalid state: not found in stored states")
	ErrEncryptionFailed      = errors.New("failed to encrypt state data")
	ErrDecryptionFailed      = errors.New("failed to decrypt state data")
	ErrSerializationFailed   = errors.New("failed to serialize state data")
	ErrDeserializationFailed = errors.New("failed to deserialize state data")
)

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

	// we encrypt using AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	paddedData := pkcs7Pad(jsonData, aes.BlockSize)
	ciphertext := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedData)

	encryptedData := append(iv, ciphertext...)
	return base64.URLEncoding.EncodeToString(encryptedData), nil
}

// DecryptState decrypts and deserializes the encrypted state
func DecryptState[T any](key []byte, state string) (string, T, error) {
	var empty T

	// base64 decode
	encryptedData, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return "", empty, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

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
