package nativemessaging

import (
	"math"
	"testing"
)

func TestFramedSize(t *testing.T) {
	t.Parallel()
	got, n32, err := framedSize(12)
	if err != nil || got != 16 || n32 != 12 {
		t.Fatalf("framedSize(12) = %d, %d, %v", got, n32, err)
	}
	if _, _, err := framedSize(-1); err == nil {
		t.Fatal("framedSize(-1): want error")
	}
	if _, _, err := framedSize(math.MaxInt); err == nil {
		t.Fatal("framedSize(MaxInt): want error")
	}
	if _, _, err := framedSize(math.MaxInt - 3); err == nil {
		t.Fatal("framedSize(MaxInt-3): want error")
	}
	u := uint64(math.MaxUint32)
	if u <= uint64(math.MaxInt) {
		got, n32, err = framedSize(int(u))
		if err != nil || got != 4+int(u) || uint64(n32) != u {
			t.Fatalf("framedSize(MaxUint32) = %d, %d, %v", got, n32, err)
		}
		if _, _, err := framedSize(int(u) + 1); err == nil {
			t.Fatal("framedSize(MaxUint32+1): want error")
		}
	}
}
