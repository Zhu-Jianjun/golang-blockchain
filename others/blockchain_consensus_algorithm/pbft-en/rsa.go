package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
)


func genRsaKeys() {
	if !isExist("./Keys") {
		fmt.Println("generating the keypair...")
		err := os.Mkdir("Keys", 0644)
		if err != nil {
			log.Panic()
		}
		for i := 0; i <= 4; i++ {
			if !isExist("./Keys/N" + strconv.Itoa(i)) {
				err := os.Mkdir("./Keys/N"+strconv.Itoa(i), 0644)
				if err != nil {
					log.Panic()
				}
			}
			priv, pub := getKeyPair()
			privFileName := "Keys/N" + strconv.Itoa(i) + "/N" + strconv.Itoa(i) + "_RSA_PIV"
			file, err := os.OpenFile(privFileName, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Panic(err)
			}
			defer file.Close()
			file.Write(priv)

			pubFileName := "Keys/N" + strconv.Itoa(i) + "/N" + strconv.Itoa(i) + "_RSA_PUB"
			file2, err := os.OpenFile(pubFileName, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				log.Panic(err)
			}
			defer file2.Close()
			file2.Write(pub)
		}
		fmt.Println("Generating key pair is done.")
	}
}

//to generate rsa key pair
func getKeyPair() (prvkey, pubkey []byte) {
	
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey = pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	pubkey = pem.EncodeToMemory(block)
	return
}


func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println(err)
		return false
	}
	return true
}

//digital signiture
func (p *pbft) RsaSignWithSha256(data []byte, keyBytes []byte) []byte {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("private key error"))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Printf("Error from signing: %s\n", err)
		panic(err)
	}

	return signature
}

//to verify the signiture
func (p *pbft) RsaVerySignWithSha256(data, signData, keyBytes []byte) bool {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("public key error"))
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signData)
	if err != nil {
		panic(err)
	}
	return true
}