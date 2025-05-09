package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"go-chat/config"
	"io"

	"golang.org/x/crypto/hkdf"
)

func Encrypt(plaintext []byte, senderPrivateKey *ecdh.PrivateKey, recipientPublicKey *ecdh.PublicKey) ([]byte, error) {
	aesKey, err := generateAESKey()
	if err != nil {
		return nil, err
	}

	encryptedMessage, err := encryptAESGCM(aesKey, plaintext)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := computeSharedSecret(senderPrivateKey, recipientPublicKey)
	if err != nil {
		return nil, err
	}

	derivedKey, err := deriveKey(sharedSecret, 32)
	if err != nil {
		return nil, err
	}

	encryptedAESKey, err := encryptAESGCM(derivedKey, aesKey)
	if err != nil {
		return nil, err
	}

	result := append(encryptedAESKey, encryptedMessage...)
	return result, nil
}

func Decrypt(ciphertext []byte, recipientPrivateKey *ecdh.PrivateKey, senderPublicKey *ecdh.PublicKey) ([]byte, error) {
	aesLen := config.AESKeyLen
	if len(ciphertext) < aesLen {
		return nil, errors.New("ciphertext too short")
	}

	encryptedAESKey := ciphertext[:aesLen]
	encryptedMessage := ciphertext[aesLen:]

	sharedSecret, err := computeSharedSecret(recipientPrivateKey, senderPublicKey)
	if err != nil {
		return nil, err
	}

	derivedKey, err := deriveKey(sharedSecret, 32)
	if err != nil {
		return nil, err
	}

	aesKey, err := decryptAESGCM(derivedKey, encryptedAESKey)
	if err != nil {
		return nil, err
	}

	plaintext, err := decryptAESGCM(aesKey, encryptedMessage)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func computeSharedSecret(privateKey *ecdh.PrivateKey, peerPublicKey *ecdh.PublicKey) ([]byte, error) {
	sharedSecret, err := privateKey.ECDH(peerPublicKey)
	if err != nil {
		return nil, err
	}
	return sharedSecret, nil
}

func generateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	// Добавляем nonce в начало результата
	result := append(nonce, ciphertext...)
	return result, nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func deriveKey(sharedSecret []byte, length int) ([]byte, error) {
	hkdfInstance := hkdf.New(sha256.New, sharedSecret, nil, nil)
	key := make([]byte, length)
	if _, err := io.ReadFull(hkdfInstance, key); err != nil {
		return nil, err
	}
	return key, nil
}
