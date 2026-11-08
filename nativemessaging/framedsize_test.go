package nativemessaging

import (
	"math"
	"testing"
)

func TestFramedSize(t *testing.T) {
	t.Parallel()
	got, err := framedSize(12)
	if err != nil || got != 16 {
		t.Fatalf("framedSize(12) = %d, %v", got, err)
	}
	if _, err := framedSize(-1); err == nil {
		t.Fatal("framedSize(-1): want error")
	}
	if _, err := framedSize(math.MaxInt); err == nil {
		t.Fatal("framedSize(MaxInt): want error")
	}
	if _, err := framedSize(math.MaxInt - 3); err == nil {
		t.Fatal("framedSize(MaxInt-3): want error")
	}
	u := uint64(math.MaxUint32)
	if u <= uint64(math.MaxInt) {
		got, err = framedSize(int(u))
		if err != nil || got != 4+int(u) {
			t.Fatalf("framedSize(MaxUint32) = %d, %v", got, err)
		}
		if _, err := framedSize(int(u) + 1); err == nil {
			t.Fatal("framedSize(MaxUint32+1): want error")
		}
	}
}
