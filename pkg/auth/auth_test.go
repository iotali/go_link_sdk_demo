package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateMQTTCredentialsAtMatchesPlatformSignature(t *testing.T) {
	credentials := GenerateMQTTCredentialsAt(
		"testProduct",
		"testDevice",
		"testSecret",
		"3",
		time.UnixMilli(1700000000000),
	)

	if credentials.Username != "testDevice&testProduct" {
		t.Fatalf("username = %q", credentials.Username)
	}
	if credentials.Password != "cf010f6c3258ec9917e7cee0dad88c964d0fd125343140b6a78e715e87da3040" {
		t.Fatalf("password = %q", credentials.Password)
	}
	if !strings.HasPrefix(credentials.ClientID, "testProduct.testDevice|timestamp=1700000000000,_ss=1,_v=4,securemode=3,signmethod=hmacsha256,ext=3,") {
		t.Fatalf("client id = %q", credentials.ClientID)
	}
}
