package idgen

import (
	"fmt"
	"math/rand"
)

// UUID is defined by RFC 4122
type UUID [16]byte

// NewUUIDv4 produces random (version 4) UUID.
func NewUUIDv4(r *rand.Rand) (UUID, error) {
	var uuid UUID
	_, err := r.Read(uuid[:16])
	if err != nil {
		return uuid, err
	}
	// variant, section 4.1.1:
	// 10xx xxxx (0x8/9/a/b)
	uuid[8] = uuid[8]&0x3f | 0x80
	// version 4, see section 4.1.3:
	// 0100 xxxx (0x4)
	uuid[6] = uuid[6]&0x0f | (4 << 4)
	return uuid, nil
}

// String returns UUID in cannonical format.
func (uuid UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
