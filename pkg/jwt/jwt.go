package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"reflect"

	"github.com/golang-jwt/jwt/v5"
)

type JwtClaims struct {
	jwt.RegisteredClaims
	Kind     int    `json:"kind,omitempty"`
	UID      uint   `json:"uid,omitempty"`
	OAuthID  string `json:"oauthid,omitempty"`
	UserName string `json:"username,omitempty"`
	Version  int    `json:"version,omitempty"`
}

func GenJwt(secret string, claims jwt.Claims) (string, error) {
	if err := validateExpiration(claims); err != nil {
		return "", err
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseJwt[T jwt.Claims, PT interface {
	jwt.Claims
	*T
}](secret string, token string, _ ...T) (*T, error) {
	return parseJwt[T, PT](token, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.SigningMethodHS256.Alg())
}

// GenRSAJwt 使用 RSA 私钥和 RS256 算法签名 Claims。
func GenRSAJwt(privateKey *rsa.PrivateKey, claims jwt.Claims) (string, error) {
	if privateKey == nil {
		return "", errors.New("RSA private key is nil")
	}
	if err := validateExpiration(claims); err != nil {
		return "", err
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return t.SignedString(privateKey)
}

// ParseRSAJwt 使用 RSA 公钥验证并解析 RS256 Token。
func ParseRSAJwt[T jwt.Claims, PT interface {
	jwt.Claims
	*T
}](publicKey *rsa.PublicKey, token string, _ ...T) (*T, error) {
	if publicKey == nil {
		return nil, errors.New("RSA public key is nil")
	}
	return parseJwt[T, PT](token, func(*jwt.Token) (any, error) {
		return publicKey, nil
	}, jwt.SigningMethodRS256.Alg())
}

// ParseRSAPrivateKey 解析采用 PKCS#1 或 PKCS#8 PEM 编码的 RSA 私钥。
func ParseRSAPrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid RSA private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RSA private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is %T, not RSA", key)
	}
	return rsaKey, nil
}

// ParseRSAPublicKey 解析采用 PKIX 或 PKCS#1 PEM 编码的 RSA 公钥。
func ParseRSAPublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid RSA public key PEM")
	}

	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RSA public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is %T, not RSA", key)
	}
	return rsaKey, nil
}

func validateExpiration(claims jwt.Claims) error {
	if claims == nil {
		return errors.New("claims are nil")
	}
	if exp, err := claims.GetExpirationTime(); err != nil || exp == nil {
		return errors.New("failed to get expiration time")
	}
	return nil
}

func parseJwt[T jwt.Claims, PT interface {
	jwt.Claims
	*T
}](token string, keyFunc jwt.Keyfunc, method string) (*T, error) {
	tTyp := reflect.TypeOf((*T)(nil)).Elem()
	tVal := reflect.New(tTyp).Interface().(jwt.Claims)
	t, err := jwt.ParseWithClaims(token, tVal, keyFunc, jwt.WithValidMethods([]string{method}))
	if err != nil {
		return nil, err
	}

	if claims, ok := t.Claims.(PT); !ok {
		return nil, fmt.Errorf("unknown claims type, cannot proceed")
	} else {
		return claims, nil
	}
}
