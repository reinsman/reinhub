package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/reinsman/reinhub/pkg/strutil"
)

type myClaims struct {
	Sub  string `json:"sub"`
	Root bool   `json:"root"`
	jwt.RegisteredClaims
}

// go test -run TestCheckJwt
func TestCheckJwt(t *testing.T) {
	key := "whoisthere"
	token := `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJpYXJuOmlhbTo6MTIzNDU2Nzg5OnVzZXIvMTIzNDUiLCJyb290Ijp0cnVlLCJqdGkiOiJrdWVkazlxeng2MzRjemd0In0.OzCWUvuTd23R_DoBCya1pOE-25wIjrTVIaQhEL-UVDQ`
	claims, err := ParseJwt(key, token, myClaims{})
	if err != nil {
		t.Error(err)
	} else {
		t.Log(claims)
	}
}

func TestGenJwt(t *testing.T) {
	key := "whoisthere"
	token, err := GenJwt(key, myClaims{
		"iarn:iam::123456789:user/12345",
		true,
		jwt.RegisteredClaims{
			ID:        strutil.Randstr(16),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)),
		},
	})
	if err != nil {
		t.Error(err)
	} else {
		t.Log(token)
	}
}

func TestRSAJwt(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	claims := myClaims{
		Sub:  "iarn:iam::123456789:user/12345",
		Root: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token, err := GenRSAJwt(privateKey, claims)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRSAJwt(&privateKey.PublicKey, token, myClaims{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Sub != claims.Sub || parsed.Root != claims.Root {
		t.Fatalf("claims = %#v, want %#v", parsed, claims)
	}
}

func TestParseRSAKeys(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	pkix, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		pem   []byte
		parse func([]byte) error
	}{
		{
			name: "PKCS#1 private key",
			pem: pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			}),
			parse: func(data []byte) error {
				_, err := ParseRSAPrivateKey(data)
				return err
			},
		},
		{
			name: "PKCS#8 private key",
			pem:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}),
			parse: func(data []byte) error {
				_, err := ParseRSAPrivateKey(data)
				return err
			},
		},
		{
			name: "PKCS#1 public key",
			pem: pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
			}),
			parse: func(data []byte) error {
				_, err := ParseRSAPublicKey(data)
				return err
			},
		},
		{
			name: "PKIX public key",
			pem:  pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}),
			parse: func(data []byte) error {
				_, err := ParseRSAPublicKey(data)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.parse(tt.pem); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestJwtRejectsUnexpectedSigningMethod(t *testing.T) {
	claims := myClaims{RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("whoisthere"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseJwt("whoisthere", token, myClaims{}); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}
}
