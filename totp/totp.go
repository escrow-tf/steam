package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"
)

type State struct {
	sharedSecret   []byte
	identitySecret []byte
}

func NewState(sharedSecret string, identitySecret string) (*State, error) {
	sharedKey, err := base64.StdEncoding.DecodeString(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("error decoding shared secret: %s", err)
	}

	identityKey, err := base64.StdEncoding.DecodeString(identitySecret)
	if err != nil {
		return nil, fmt.Errorf("error decoding identity secret: %s", err)
	}

	return &State{
		sharedSecret:   sharedKey,
		identitySecret: identityKey,
	}, nil
}

func Time(offset int64) time.Time {
	return time.Now().UTC().Add(time.Second * time.Duration(offset))
}

func (s State) GenerateTotpCode(tag string, time time.Time) (string, error) {
	// Converting time for any reason
	// 00 00 00 00 00 00 00 00
	// 00 00 00 00 xx xx xx xx
	unixTime := uint64(time.Unix()) / 30
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, unixTime)

	// Evaluate hash code for `tb` by key
	mac := hmac.New(sha1.New, s.sharedSecret)
	mac.Write(timeBytes)
	hashcode := mac.Sum(nil)

	// Last 4 bits provide initial position
	// len(hashcode) = 20 bytes
	start := hashcode[19] & 0xf

	// Extract 4 bytes at `start` and drop first bit
	fc32 := binary.BigEndian.Uint32(hashcode[start : start+4])
	fc32 &= 1<<31 - 1
	fullCode := int(fc32)

	// Range of possible chars for auth code.
	//goland:noinspection SpellCheckingInspection
	var chars = "23456789BCDFGHJKMNPQRTVWXY"
	var charsLen = len(chars)

	// Generate auth code
	code := make([]byte, 5)
	for i := range code {
		code[i] = chars[fullCode%charsLen]
		fullCode /= charsLen
	}

	return string(code[:]), nil
}

func (s State) GenerateConfirmationKey(useTime time.Time, tag []byte) (hashCode []byte, err error) {
	// the input buffer is
	dataLength := 8
	if len(tag) != 0 {
		if len(tag) > 32 {
			dataLength += 32
		} else {
			dataLength += len(tag)
		}
	}

	buffer := make([]byte, dataLength)
	binary.BigEndian.PutUint64(buffer, uint64(useTime.Unix()))

	if len(tag) != 0 {
		copy(buffer[8:], tag)
	}

	hmacHash := hmac.New(sha1.New, s.identitySecret)
	if _, err = hmacHash.Write(buffer); err != nil {
		return
	}
	hashCode = hmacHash.Sum(nil)
	return
}

func GetDeviceId(steamID string) string {
	checksum := sha1.Sum([]byte(steamID))
	checksumBase64 := base64.StdEncoding.EncodeToString(checksum[:])
	return fmt.Sprintf("android:%s", checksumBase64)
}
