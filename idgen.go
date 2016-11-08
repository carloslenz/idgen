// Package idgen implements algorithms for generation of IDs.
package idgen

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type (
	// Interface provides ID generation. There are are a few implementations available.
	Interface interface {
		// NewIDs generates n new IDs. The last ID is returned, so for the first subtract
		// n-1. Each implementation may specify the maximum n it accepts, otherwise an
		// error is returned. Otherwise noted, overflows and clashes are not checked.
		NewIDs(n int64) (int64, error)
	}
)

// NewSnowflake returns an ID generator that follows Twitter's Snowflake algorithm.
// It provides up to 4096 IDs per millisecond (so it tries to avoid clashes if possible),
// supports up to 1024 generating nodes until year 2038. One bit is ignored, which could
// be used to extend the maximum date further.
func NewSnowflake(nodeMask int64) Interface {
	seq := &sequential{}
	return &snowflake{
		// Needed to reset when a new timestamp is entered.
		sequential: seq,
		// Least significant bits: only one that accepts counter > 1.
		seqChecker: NewOverflowChecker(12, seq),
		// TODO: does not check nodeMask overflow up-front.
		constant: shifted{
			gen:  NewOverflowChecker(10, constant(nodeMask)),
			bits: 12,
		},
		tstamp: shifted{
			gen:  NewOverflowChecker(41, NewTimestamp()),
			bits: 22,
		},
	}
}

// NewSequential returns an ID generator with reproducible results, so it is suitable for
// tests.
func NewSequential() Interface {
	return &sequential{}
}

// NewNegSequential returns an ID generator with reproducible results which starts from
// the lowest possible int64, so it is good for migrations, so when the system becomes
// active another generator is used, while clashes are avoided implicitly if it generates
// positive IDs.
func NewNegSequential() Interface {
	return &sequential{value: int64(-(1 << 63))}
}

// NewOverflowChecker wraps an ID generator to check for overflows.
func NewOverflowChecker(allowedBits byte, gen Interface) Interface {
	return overflowChecker{
		gen:          gen,
		overflowBits: ^(1<<allowedBits - 1),
	}
}

// NewTimestamp returns an ID generator that uses the machine clock (Millisec precision).
// Not safe for concurrent use by itself (neither does it check for clashes), since
// resolution is in Milliseconds.
func NewTimestamp() Interface {
	return tstamp{}
}

// Implementation
// ==============

type (
	// constant is used to implement the nodeMask in Snowflake.
	// It cannot generated more than one ID at once.
	constant   int64
	sequential struct {
		value int64
	}
	tstamp          struct{}
	overflowChecker struct {
		gen          Interface
		overflowBits int64
	}
	// shifted wraps another generator to left-shift its bits.
	shifted struct {
		gen  Interface
		bits byte
	}
	snowflake struct {
		sync.Mutex
		lastTimestamp int64
		tstamp        Interface
		constant      Interface
		seqChecker    Interface
		sequential    *sequential
	}
)

// NewIDs always returns the same constants, and only accepts n=1.
func (c constant) NewIDs(n int64) (int64, error) {
	if err := checkNIsOne(c, 1, n); err != nil {
		return 0, err
	}
	return int64(c), nil
}

// NewIDs uses the machine clock and it only accepts n=1.
// Not safe for concurrent use by itself, since resolution is in Milliseconds.
func (t tstamp) NewIDs(n int64) (int64, error) {
	if err := checkNIsOne(t, 1, n); err != nil {
		return 0, err
	}
	return time.Now().UnixNano() / int64(time.Millisecond), nil
}

// NewIDs executes the wrapped generator and checks for overflow.
func (o overflowChecker) NewIDs(n int64) (int64, error) {
	v, err := o.gen.NewIDs(n)
	if err != nil {
		return 0, err
	}
	if bits := v & o.overflowBits; bits != 0 {
		return 0, fmt.Errorf("%T.NewIDs() overflow %b", o.gen, bits)
	}
	return v, nil
}

// NewIDs executes the wrapped generator and shifts its result.
func (s shifted) NewIDs(n int64) (int64, error) {
	v, err := s.gen.NewIDs(n)
	if err != nil {
		return 0, err
	}
	return v << s.bits, nil
}

// NewIDs ands n to the internal counter and returns it. It's safe for concurrent use.
func (s *sequential) NewIDs(n int64) (int64, error) {
	return atomic.AddInt64(&s.value, n), nil
}

// reset is used by snowflake to restart the sequence when the timestamp changes.
func (s *sequential) reset(v int64) {
	atomic.StoreInt64(&s.value, v)
}

// NewIDs combines a timestamp, a (constant) nodeMask and a sequence.
// Safe for concurrent use.
func (s *snowflake) NewIDs(n int64) (int64, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	var err error
	var tstamp, nodeMask, seqNum int64
	if tstamp, err = s.tstamp.NewIDs(1); err != nil {
		return 0, err
	}

	if tstamp != s.lastTimestamp {
		seqNum = n - 1
		s.sequential.reset(seqNum)
		s.lastTimestamp = tstamp
	} else if seqNum, err = s.seqChecker.NewIDs(n); err != nil {
		return 0, err
	}

	if nodeMask, err = s.constant.NewIDs(1); err != nil {
		return 0, err
	}

	return tstamp | nodeMask | seqNum, nil
}

func checkNIsOne(gen Interface, max, n int64) error {
	if n != max {
		return fmt.Errorf("%T/%v.NewIDs() supports count=%v, got %v",
			gen, gen, max, n)
	}
	return nil
}
