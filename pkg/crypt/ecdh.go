package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

func Encrypt(
	in []byte,
	senderKey *ecdh.PrivateKey,
	receiverKey *ecdh.PublicKey,
) ([]byte, error) {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}
	//
	// Создание AES-GCM для шифрования данных
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Генерация nonce для AES-GCM
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Шифрование данных
	encryptedPayload := gcm.Seal(nonce, nonce, in, nil)

	// Получение общего секрета ECDH
	sharedSecret, err := senderKey.ECDH(receiverKey)
	if err != nil {
		return nil, err
	}
	sharedKey := sha256.Sum256(sharedSecret)

	// Шифрование ключа AES с помощью XOR
	encryptedAESKey := make([]byte, 32)
	for i := range aesKey {
		encryptedAESKey[i] = aesKey[i] ^ sharedKey[i]
	}

	return append(encryptedAESKey, encryptedPayload...), nil
}

func Decrypt(in []byte, receiverKey *ecdh.PrivateKey, senderKey *ecdh.PublicKey) ([]byte, error) {
	if len(in) < 44 { // 32 (encryptedAESKey) + 12 (nonce) минимум
		return nil, errors.New("input too short")
	}

	// Извлекаем зашифрованный ключ AES и зашифрованный текст
	pos := 0
	encryptedAESKey := in[pos : pos+32]
	pos += 32
	encryptedPayload := in[pos:]

	// Получение общего секрета ECDH
	sharedSecret, err := receiverKey.ECDH(senderKey)
	if err != nil {
		return nil, err
	}
	sharedKey := sha256.Sum256(sharedSecret)

	// Расшифровка ключа AES с помощью XOR
	aesKey := make([]byte, 32)
	for i := range aesKey {
		aesKey[i] = encryptedAESKey[i] ^ sharedKey[i]
	}

	// Создание AES-GCM для расшифровки данных
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Извлекаем nonce из начала зашифрованного текста
	nonceSize := gcm.NonceSize()
	if len(encryptedPayload) < nonceSize {
		return nil, errors.New("invalid encrypted payload")
	}
	nonce := encryptedPayload[:nonceSize]
	ciphertext := encryptedPayload[nonceSize:]

	// Расшифровка данных
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
