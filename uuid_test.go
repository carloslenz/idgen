package idgen

import (
	"math/rand"
	"testing"
)

func TestUUID(t *testing.T) {
	r := rand.New(rand.NewSource(137))
	ids := []struct {
		v [16]byte
		s string
	}{
		{[...]byte{0x66, 0x49, 0xea, 0x0e, 0x18, 0xca, 0x46, 0xf4, 0x84, 0x01, 0xbf, 0x55, 0xfb,
			0xe1, 0x4a, 0x1b}, "6649ea0e-18ca-46f4-8401-bf55fbe14a1b"},
		{[...]byte{0xd1, 0x5d, 0x11, 0x07, 0xc6, 0x8f, 0x49, 0x27, 0x8f, 0x62, 0x99, 0x31, 0xb4,
			0x41, 0xdc, 0x85}, "d15d1107-c68f-4927-8f62-9931b441dc85"},
	}

	for i, id := range ids {
		uuid, err := NewUUIDv4(r)
		s := uuid.String()
		switch {
		case err != nil:
			t.Errorf("%d: %s", i, err)
		case uuid != id.v:
			t.Errorf("%d: UUID, got %v, expected %v", i, uuid, id.v)
		case s != id.s:
			t.Errorf("%d: repr, got %s, expected %s", i, s, id.s)
		}

	}
}
