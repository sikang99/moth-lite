// =================================================================================
// Filename: util-crypto.go
// Function: Crypto functions
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2022
// =================================================================================
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"time"
)

// ---------------------------------------------------------------------------------
func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func RandomId(n int) (str string) {
	b := make([]byte, n)

	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		log.Println(err)
		return
	}
	str = hex.EncodeToString(b)
	return
}

// ---------------------------------------------------------------------------------
// [Learn Golang encryption and decryption](https://blog.logrocket.com/learn-golang-encryption-decryption/)
// [AES-256-GCM in Golang](https://jusths.tistory.com/232)
// https://github.com/nicewook/golang-aes-gsm-example
// ---------------------------------------------------------------------------------
func AES256GSMEncrypt(secretKey, secondKey, plaintext []byte) (ciphertext []byte, err error) {
	if len(secretKey) != 32 {
		err = fmt.Errorf("secret key is not for AES-256: total %d bits", 8*len(secretKey))
		return
	}

	// prepare AES-256-GSM cipher
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err

	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// make random nonce
	nonce := make([]byte, aesgcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		log.Println(err)
		return
	}

	// encrypt plaintext with second key
	ciphertext = aesgcm.Seal(nonce, nonce, plaintext, secondKey)

	// log.Printf("[Encrypt] nonce: %x, ciphertext: %x\n", nonce, ciphertext)
	return ciphertext, nil
}

// ---------------------------------------------------------------------------------
func AES256GSMDecrypt(secretKey []byte, ciphertext []byte, secondKey []byte) (plaintext []byte, err error) {
	if len(secretKey) != 32 {
		err = fmt.Errorf("secret key is not for AES-256: total %d bits", 8*len(secretKey))
		return
	}

	// prepare AES-256-GSM cipher
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		log.Println(err)
		return
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Println(err)
		return
	}

	nonceSize := aesgcm.NonceSize()
	nonce, pureCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// decrypt ciphertext with second key
	plaintext, err = aesgcm.Open(nil, nonce, pureCiphertext, secondKey)
	if err != nil {
		log.Println(err)
		return
	}

	// log.Printf("[Decrypt] ciphertext: %x, nonce: %x, plaintext: %x\n", ciphertext, nonce, plaintext)
	return
}

// ---------------------------------------------------------------------------------
// [An Introduction to Cryptography in Go](https://www.developer.com/languages/cryptography-in-go/)
// ---------------------------------------------------------------------------------
func encryptMessage(key string, message string) (str string) {
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Println(err)
		return
	}
	msgByte := make([]byte, len(message))
	c.Encrypt(msgByte, []byte(message))
	str = hex.EncodeToString(msgByte)
	return
}

// ---------------------------------------------------------------------------------
func decryptMessage(key string, message string) (msg string) {
	txt, _ := hex.DecodeString(message)
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		log.Println(err)
		return
	}
	msgByte := make([]byte, len(txt))
	c.Decrypt(msgByte, []byte(txt))
	msg = string(msgByte[:])
	return
}

// ---------------------------------------------------------------------------------
// [RSA Encryption And Decryption Example](https://www.knowledgefactory.net/2021/12/go-rsa-encryption-and-decryption-example.html)
func EncryptRSAWithPublicKey(secretMessage string, key rsa.PublicKey) (str string) {
	// Encryption with OAEP padding
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha512.New(), rng, &key, []byte(secretMessage), nil)
	if err != nil {
		log.Println(err)
		return
	}

	str = base64.StdEncoding.EncodeToString(ciphertext)
	return
}

// ---------------------------------------------------------------------------------
func DecryptRSAWithPrivateKey(cipherText string, privKey rsa.PrivateKey) (str string) {
	// Decode the Cipher text
	ct, err := base64.StdEncoding.DecodeString(cipherText)
	rng := rand.Reader
	secrettext, err := rsa.DecryptOAEP(sha512.New(), rng, &privKey, ct, nil)
	if err != nil {
		log.Println(err)
		return
	}

	str = string(secrettext)
	return
}

// ---------------------------------------------------------------------------------
func GenerateCertPair(start, end time.Time) (caCert *x509.Certificate, caPrivateKey *ecdsa.PrivateKey, err error) {
	b := make([]byte, 8)
	_, err = rand.Read(b)
	if err != nil {
		log.Println(err)
		return
	}
	serial := int64(binary.BigEndian.Uint64(b))
	if serial < 0 {
		serial = -serial
	}
	certTempl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{},
		NotBefore:             start,
		NotAfter:              end,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Println(err)
		return
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, certTempl, certTempl, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		log.Println(err)
		return
	}
	caCert, err = x509.ParseCertificate(caBytes)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
// GenerateTLSConf generates a CA certificate and private key for server.
// https://github.com/marten-seemann/webtransport-go/blob/master/interop/main.go
// ---------------------------------------------------------------------------------
func GetTLSConf(start, end time.Time) (*tls.Config, error) {
	cert, priv, err := GenerateTLSCert(start, end)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{cert.Raw},
			PrivateKey:  priv,
			Leaf:        cert,
		}},
	}, nil
}

// ---------------------------------------------------------------------------------
func GenerateTLSCert(start, end time.Time) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return nil, nil, err
	}
	serial := int64(binary.BigEndian.Uint64(b))
	if serial < 0 {
		serial = -serial
	}
	certTempl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{},
		NotBefore:             start,
		NotAfter:              end,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, certTempl, certTempl, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	ca, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, nil, err
	}
	return ca, caPrivateKey, nil
}

// ---------------------------------------------------------------------------------
func ReadPEMFile(fname string) (err error) {
	r, err := os.ReadFile(fname)
	if err != nil {
		log.Println(err)
		return
	}

	block, _ := pem.Decode(r)
	log.Println(block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(cert.Subject)

	hash := sha256.Sum256(cert.Raw)
	log.Println(hash)
	return
}

//=================================================================================
