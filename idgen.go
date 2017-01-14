// Package idgen implements algorithms for generation of IDs.
package idgen

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type (
	// Interface provides ID generation. There are a few implementations available.
	Interface interface {
		// NewIDs generates n new IDs. The last ID is returned, so for the first ID subtract
		// n-1. Each implementation may specify the maximum accepted n (otherwise an
		// error is returned). Overflows and clashes are not checked unless an implementation
		// says so.
		NewIDs(n int64) (int64, error)
	}
)

// NewSnowflake returns an ID generator that follows Twitter's Snowflake algorithm.
// It can generate up to 4096 IDs per millisecond (so it tries to avoid clashes if possible),
// and supports up to 1024 generating nodes, up until year 2038. ID's Leading bit is always 0
// so the returned ID is never negative (i.e, 63 of 64 bits are significative).
// Safe for concurrent use.
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
// the lowest possible int64. It is useful for migrations, so when the system becomes
// active and a concurrent generator is used, clashes are avoided implicitly
// (since the following generator would only produce positive IDs).
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

// NewTimestamp returns an ID generator that uses the machine clock (Millisecond precision).
// So it is safe for concurrent use by itself (neither does it check for clashes).
// It's NewIDs method only accepts n=1.
func NewTimestamp() Interface {
	return tstamp{}
}

// Implementation
// ==============

type (
	// constant is used to implement the nodeMask in Snowflake.
	// It cannot generated more than one ID at once.
	constant int64
	// sequential generates IDs by adding to internal counter. It's safe for concurrent use.
	sequential struct {
		value int64
	}
	tstamp struct{}
	// overflowChecker executes gen and checks for overflow.
	overflowChecker struct {
		gen          Interface
		overflowBits int64
	}
	// shifted executen gen and left-shifts the generated ID's bits.
	shifted struct {
		gen  Interface
		bits byte
	}
	// snowflake combines a timestamp, a (constant) nodeMask and a sequence.
	snowflake struct {
		sync.Mutex
		lastTimestamp int64
		tstamp        Interface
		constant      Interface
		seqChecker    Interface
		sequential    *sequential
	}
)

func (c constant) NewIDs(n int64) (int64, error) {
	if err := checkNIsOne(c, n); err != nil {
		return 0, err
	}
	return int64(c), nil
}

func (t tstamp) NewIDs(n int64) (int64, error) {
	if err := checkNIsOne(t, n); err != nil {
		return 0, err
	}
	return time.Now().UnixNano() / int64(time.Millisecond), nil
}

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

func (s shifted) NewIDs(n int64) (int64, error) {
	v, err := s.gen.NewIDs(n)
	if err != nil {
		return 0, err
	}
	return v << s.bits, nil
}

func (s *sequential) NewIDs(n int64) (int64, error) {
	return atomic.AddInt64(&s.value, n), nil
}

// reset is used by snowflake to restart the sequence when the timestamp changes.
func (s *sequential) reset(v int64) {
	atomic.StoreInt64(&s.value, v)
}

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

func checkNIsOne(gen Interface, n int64) error {
	if n != 1 {
		return fmt.Errorf("%T/%v.NewIDs() supports count=1, got %v",
			gen, gen, n)
	}
	return nil
}
