package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const PASSWORD_ENCRYT_KEY = "2019112300000000"

// https://www.devglan.com/online-tools/aes-encryption-decryption
// key 默认16字节
// iv 默认是key
func AesEncryptCBC(orig string, key string) string {
	origData := []byte(orig)
	k := []byte(key)

	block, _ := aes.NewCipher(k)

	blockSize := block.BlockSize()

	origData = PKCS7Padding(origData, blockSize)

	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])

	cryted := make([]byte, len(origData))

	blockMode.CryptBlocks(cryted, origData)
	return base64.StdEncoding.EncodeToString(cryted)
}

func AesDecryptCBC(cryted string, key string) string {
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
	k := []byte(key)

	block, _ := aes.NewCipher(k)

	blockSize := block.BlockSize()

	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])

	orig := make([]byte, len(crytedByte))

	blockMode.CryptBlocks(orig, crytedByte)

	orig = PKCS7UnPadding(orig)
	return string(orig)
}
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padText...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func hmac_sha256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HashMethod 表示密码哈希方法。
type HashMethod string

const (
	HM_SHA256       HashMethod = "sha256"
	HM_PBKDF2SHA256 HashMethod = "pbkdf2:sha256"
	HM_PLAIN        HashMethod = "plain"
)
const (
	keyLength = 32
)

func GeneratePasswordHash(password string, method ...HashMethod) string {
	salt := ""
	m := HM_SHA256
	if len(method) > 0 {
		m = method[0]
	}

	if m != HM_PLAIN {
		salt = gen_salt()
	}

	h, actual_method := hash_internal(m, salt, password)
	return fmt.Sprintf("%s$%s$%s", actual_method, salt, h)
}

func CheckPasswordHash(password string, hash string) bool {
	if strings.Count(hash, "$") < 2 {
		return false
	}
	ps := strings.Split(hash, "$")
	h, _ := hash_internal(HashMethod(ps[0]), ps[1], password)
	return ps[2] == h
}

func gen_salt() string {
	satl_chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	salt_len := 8
	var bytes = make([]byte, salt_len)
	rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = satl_chars[v%byte(len(satl_chars))]
	}
	return string(bytes)
}

// hash_internal 返回哈希结果和实际使用的方法。
func hash_internal(method HashMethod, salt, password string) (string, string) {
	if method == HM_SHA256 {
		return hmac_sha256(password, salt), string(method)
	}

	if method == HM_PLAIN {
		return password, string(method)
	}

	{
		m := string(method)
		if strings.HasPrefix(m, string(HM_PBKDF2SHA256)) {
			iters := 150000
			fields := strings.Split(m, ":")
			if len(fields) == 3 {
				iters, _ = strconv.Atoi(fields[2])
			}
			hash := pbkdf2.Key([]byte(password), []byte(salt), iters, keyLength, sha256.New)
			return hex.EncodeToString(hash), fmt.Sprintf("%s:%d", string(method), iters)
		}
	}
	panic(fmt.Sprintf("unknown hash method: %v", method))
}
