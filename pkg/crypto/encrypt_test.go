package crypto

import (
	"testing"
)

// go test -run TestCheckPasswordHash
func TestCheckPasswordHash(t *testing.T) {
	actual := CheckPasswordHash("password", "pbkdf2:sha256:150000$myj7pNIO$895b262253b5b2944ba342340d0e9c08100b40a5218cd9e944ed72b43cabf397")
	if actual != true {
		t.Error("CheckPasswordHash()1 return false")
	}

	actual = CheckPasswordHash("Y?@day+1ri", `sha256$Ps9yVOeI$1e33882c18dc396f46c15797bec195bd7b376c24a8bbdf1f69ab5a501d6a9fa5`)
	if actual != true {
		t.Error("CheckPasswordHash()2 return false")
	}

	actual = CheckPasswordHash("qwer1234", "pbkdf2:sha256:150000$tXdmN7QD$c153b201a4709b9e41340f284a9c4dc82cf078463dc4c6c8cc1fad48bb557f28")
	if actual != true {
		t.Error("CheckPasswordHash()3 return false")
	}

	actual = CheckPasswordHash("abcdefg", "plain$$abcdefg")
	if actual != true {
		t.Error("CheckPasswordHash()3 return false")
	}
}
