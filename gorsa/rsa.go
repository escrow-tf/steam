package gorsa

import (
	"bytes"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type RSASecurity struct {
	publicString  string          //公钥字符串
	privateString string          //私钥字符串
	publicKey     *rsa.PublicKey  //公钥
	privateKey    *rsa.PrivateKey //私钥
}

// SetPublicKey 设置公钥
func (r *RSASecurity) SetPublicKey(pubStr string) (err error) {
	r.publicString = pubStr
	r.publicKey, err = r.GetPublickey()
	return err
}

// SetPrivateKey 设置私钥
func (r *RSASecurity) SetPrivateKey(priStr string) (err error) {
	r.privateString = priStr
	r.privateKey, err = r.GetPrivateKey()
	return err
}

// GetPrivateKey *rsa.PublicKey
func (r *RSASecurity) GetPrivateKey() (*rsa.PrivateKey, error) {
	return getPriKey([]byte(r.privateString))
}

// GetPublickey *rsa.PrivateKey
func (r *RSASecurity) GetPublickey() (*rsa.PublicKey, error) {
	return getPubKey([]byte(r.publicString))
}

func EncryptPKCS1(input []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New(`please set the public key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := pubKeyIO(publicKey, bytes.NewReader(input), output, true)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(output)
}

// PubKeyENCRYPT 公钥加密
func (r *RSASecurity) PubKeyENCRYPT(input []byte) ([]byte, error) {
	if r.publicKey == nil {
		return []byte(""), errors.New(`please set the public key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := pubKeyIO(r.publicKey, bytes.NewReader(input), output, true)
	if err != nil {
		return []byte(""), err
	}
	return io.ReadAll(output)
}

// PubKeyDECRYPT 公钥解密
func (r *RSASecurity) PubKeyDECRYPT(input []byte) ([]byte, error) {
	if r.publicKey == nil {
		return []byte(""), errors.New(`please set the public key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := pubKeyIO(r.publicKey, bytes.NewReader(input), output, false)
	if err != nil {
		return []byte(""), err
	}
	return io.ReadAll(output)
}

// PriKeyENCTYPT 私钥加密
func (r *RSASecurity) PriKeyENCTYPT(input []byte) ([]byte, error) {
	if r.privateKey == nil {
		return []byte(""), errors.New(`please set the private key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := priKeyIO(r.privateKey, bytes.NewReader(input), output, true)
	if err != nil {
		return []byte(""), err
	}
	return io.ReadAll(output)
}

// PriKeyDECRYPT 私钥解密
func (r *RSASecurity) PriKeyDECRYPT(input []byte) ([]byte, error) {
	if r.privateKey == nil {
		return []byte(""), errors.New(`please set the private key in advance`)
	}
	output := bytes.NewBuffer(nil)
	err := priKeyIO(r.privateKey, bytes.NewReader(input), output, false)
	if err != nil {
		return []byte(""), err
	}

	return io.ReadAll(output)
}

// SignMd5WithRsa /**
func (r *RSASecurity) SignMd5WithRsa(data string) (string, error) {
	md5Hash := md5.New()
	sData := []byte(data)
	md5Hash.Write(sData)
	hashed := md5Hash.Sum(nil)

	signByte, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.MD5, hashed)
	sign := base64.StdEncoding.EncodeToString(signByte)
	return sign, err
}

// SignSha1WithRsa /**
func (r *RSASecurity) SignSha1WithRsa(data string) (string, error) {
	sha1Hash := sha1.New()
	sData := []byte(data)
	sha1Hash.Write(sData)
	hashed := sha1Hash.Sum(nil)

	signByte, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA1, hashed)
	sign := base64.StdEncoding.EncodeToString(signByte)
	return sign, err
}

// SignSha256WithRsa /**
func (r *RSASecurity) SignSha256WithRsa(data string) (string, error) {
	sha256Hash := sha256.New()
	sData := []byte(data)
	sha256Hash.Write(sData)
	hashed := sha256Hash.Sum(nil)

	signByte, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hashed)
	sign := base64.StdEncoding.EncodeToString(signByte)
	return sign, err
}

// VerifySignMd5WithRsa /**
func (r *RSASecurity) VerifySignMd5WithRsa(data string, signData string) error {
	sign, err := base64.StdEncoding.DecodeString(signData)
	if err != nil {
		return err
	}
	hash := md5.New()
	hash.Write([]byte(data))
	return rsa.VerifyPKCS1v15(r.publicKey, crypto.MD5, hash.Sum(nil), sign)
}

// VerifySignSha1WithRsa /**
func (r *RSASecurity) VerifySignSha1WithRsa(data string, signData string) error {
	sign, err := base64.StdEncoding.DecodeString(signData)
	if err != nil {
		return err
	}
	hash := sha1.New()
	hash.Write([]byte(data))
	return rsa.VerifyPKCS1v15(r.publicKey, crypto.SHA1, hash.Sum(nil), sign)
}

// VerifySignSha256WithRsa /**
func (r *RSASecurity) VerifySignSha256WithRsa(data string, signData string) error {
	sign, err := base64.StdEncoding.DecodeString(signData)
	if err != nil {
		return err
	}
	hash := sha256.New()
	hash.Write([]byte(data))

	return rsa.VerifyPKCS1v15(r.publicKey, crypto.SHA256, hash.Sum(nil), sign)
}
