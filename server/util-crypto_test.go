// =================================================================================
// Filename: util-crypto_test.go
// Function: Test functions for util-crypto.go
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2022
// =================================================================================
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------------
func TestAES256GSM(t *testing.T) {
	var (
		passphraseForSecretKey = "this is my secret key passphrase"
		plaintext              = "this should be encrypted"
		secondKey              = "this will be in the code which will not exposed outside"
	)

	// generate 32 byte secret key
	hash := sha256.New()
	_, err := hash.Write([]byte(passphraseForSecretKey))
	if err != nil {
		t.Error(err)
	}

	secretKey := hash.Sum(nil)
	fmt.Printf("secret key generated: %x\n", secretKey)

	ciphertext, err := AES256GSMEncrypt(secretKey, []byte(secondKey), []byte(plaintext))
	if err != nil {
		t.Error(err)
	}

	plaintextBytes, err := AES256GSMDecrypt(secretKey, ciphertext, []byte(secondKey))
	if err != nil {
		t.Error(err)
	}

	if plaintext != string(plaintextBytes) {
		t.Errorf("plaintext %s is differ from decrypted cipertext %s", plaintext, string(plaintextBytes))
	}
}

// ---------------------------------------------------------------------------------
func TestAESCryption(t *testing.T) {
	plaintext := "This is a secret"
	key := "this_must_be_of_32_byte_length!!"

	emesg := encryptMessage(key, plaintext)
	dmesg := decryptMessage(key, emesg)

	if plaintext != dmesg {
		t.Errorf("plaintext %s != %s", plaintext, dmesg)
	}
}

// ---------------------------------------------------------------------------------
func TestRSACryption(t *testing.T) {
	// 3072 is the number of bits for RSA
	bitSize := 3072
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		t.Error(err)
	}

	publicKey := privateKey.PublicKey
	plainMessage := "My Secret Text"
	cipherMessage := EncryptRSAWithPublicKey(plainMessage, publicKey)
	decodeMessage := DecryptRSAWithPrivateKey(cipherMessage, *privateKey)

	if plainMessage != decodeMessage {
		t.Errorf("plaintext %s != %s", plainMessage, decodeMessage)
	}
}

//=================================================================================
