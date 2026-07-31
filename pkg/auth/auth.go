package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Credentials struct {
	ClientID string
	Username string
	Password string
}

func GenerateMQTTCredentials(productKey, deviceName, deviceSecret, secureMode string) *Credentials {
	return GenerateMQTTCredentialsAt(productKey, deviceName, deviceSecret, secureMode, time.Now())
}

func GenerateMQTTCredentialsAt(productKey, deviceName, deviceSecret, secureMode string, now time.Time) *Credentials {
	timestamp := fmt.Sprintf("%d", now.UnixMilli())
	nonce := fmt.Sprintf("%d", now.UnixNano())

	clientID := fmt.Sprintf("%s.%s|timestamp=%s,_ss=1,_v=4,securemode=%s,signmethod=hmacsha256,ext=3,%s|",
		productKey, deviceName, timestamp, secureMode, nonce)

	username := fmt.Sprintf("%s&%s", deviceName, productKey)

	signContent := fmt.Sprintf("clientId%s.%sdeviceName%sproductKey%stimestamp%s",
		productKey, deviceName, deviceName, productKey, timestamp)

	password := calculateHMACSHA256(signContent, deviceSecret)

	return &Credentials{
		ClientID: clientID,
		Username: username,
		Password: password,
	}
}

func GenerateMQTTCredentialsLegacy(productKey, deviceName, deviceSecret string) *Credentials {
	return GenerateMQTTCredentials(productKey, deviceName, deviceSecret, "2")
}

func GenerateDynRegSignature(productKey, deviceName, productSecret, random string) string {
	signContent := fmt.Sprintf("deviceName%sproductKey%srandom%s", deviceName, productKey, random)
	return calculateHMACSHA256(signContent, productSecret)
}

func calculateHMACSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
