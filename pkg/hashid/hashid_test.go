package hashid

import (
	"testing"
)

// go test -v ./pkg/hashid/ -run TestHashIds
func TestHashIds(t *testing.T) {
	id := EnHashId(99999999)
	num, err := DeHashId(id)

	t.Log("id:", id)
	if err != nil {
		t.Fatal(err)
	}
	if num != 99999999 {
		t.Errorf("HashId roundtrip failed: got %d, want %d", num, 99999999)
	}
}
