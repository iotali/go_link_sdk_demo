package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Credentials struct {
	ClientID string
	Username string
	Password string
}

func GenerateMQTTCredentials(productKey, deviceName, deviceSecret, secureMode string) *Credentials {
	timestamp := "2524608000000" // 固定时间戳，与C SDK保持一致
	sdkVersion := "sdk-go-4.2.0"  // 使用与C SDK类似的版本格式
	
	clientID := fmt.Sprintf("%s.%s|timestamp=%s,_ss=1,_v=%s,securemode=%s,signmethod=hmacsha256,ext=3,_conn=tl|",
		productKey, deviceName, timestamp, sdkVersion, secureMode)
	
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