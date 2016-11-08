package idgen

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"
)

func TestConstant(t *testing.T) {
	t.Parallel()
	expected := int64(103)
	gen := constant(expected)
	for count := int64(0); count < 2; count++ {
		v, err := gen.NewIDs(count)
		switch {
		case count == 1 && err != nil:
			t.Errorf("TestConstant %d: got error %q", count, err)
		case count == 1 && v != expected:
			t.Errorf("TestConstant %d: expected %v, got %v", expected, count, v)
		case count != 1 && err == nil:
			t.Errorf("TestConstant %d: got %v, expected error %v", count, expected, err)
		default:
			// success
		}
	}
}

func TestSequential(t *testing.T) {
	t.Parallel()
	gen := NewSequential()
	expected := int64(0)
	for count := int64(1); count < 5; count++ {
		expected += count
		v, err := gen.NewIDs(count)
		switch {
		case err != nil:
			t.Errorf("TestSequential %d: got error %q", count, err)
			break
		case v != expected:
			t.Errorf("TestSequential %d: expected %d, got %v", count, expected, v)
			break
		}
	}
}

func TestNegSequential(t *testing.T) {
	t.Parallel()
	gen := NewNegSequential()
	expected := int64(-(1 << 63))
	for count := int64(1); count < 5; count++ {
		expected += count
		v, err := gen.NewIDs(count)
		switch {
		case err != nil:
			t.Errorf("TestSequential %d: got error %q", count, err)
			break
		case v != expected:
			t.Errorf("TestSequential %d: expected %d, got %v", count, expected, v)
			break
		}
	}
}

func TestTimestamp(t *testing.T) {
	count := int64(0)
	gen := NewTimestamp()
	if v, err := gen.NewIDs(count); err == nil {
		t.Errorf("TestTimestamp %d: expected error, got value %v", count, v)
	}
	for count++; count < 10; count++ {
		v, err := gen.NewIDs(count)
		expected := time.Now().UnixNano() / int64(time.Millisecond)
		switch {
		case count == 1 && err != nil:
			t.Errorf("TestTimestamp %d: got error %q", count, err)
		case count == 1 && math.Abs(float64(v-expected)) > 1.:
			t.Errorf("TestTimestamp %d: expected %v, got %v", expected, expected, v)
		case count != 1 && err == nil:
			t.Errorf("TestTimestamp %d: got %v, expected error %v", count, expected, err)
		default:
			// success
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func TestOverflowChecker(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		allowedBits byte
		count       int64
		expectError bool
		expected    int64
	}{
		{0, 1, true, 0},
		{1, 1, false, 1},
		{1, 2, true, 0},
		{4, 4, false, 4},
	}
	gen := repeat{}
	for i, test := range tests {
		v, err := NewOverflowChecker(test.allowedBits, gen).NewIDs(test.count)
		switch {
		case test.expectError && err == nil:
			t.Errorf("TestShifted %d: expected error, got value %v", i, v)
		case !test.expectError && err != nil:
			t.Errorf("TestShifted %d: got error %q, expected value %v",
				i, err, test.expected)
		case !test.expectError && v != test.expected:
			t.Errorf("TestShifted %d: got %v, expected %v", i, v, test.expected)
		default:
			// success
		}
	}
}

func TestShifted(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		bits     byte
		count    int64
		expected int64
	}{
		{0, 1, 1},
	}
	gen := repeat{}
	for i, test := range tests {
		sh := shifted{
			gen:  gen,
			bits: test.bits,
		}
		v, err := sh.NewIDs(test.count)
		switch {
		case err != nil:
			t.Errorf("TestShifted %d: got error %q", i, err)
		case v != test.expected:
			t.Errorf("TestShifted %d: got %v, expected %v", i, v, test.expected)
		default:
			// success
		}
	}
}

func TestErrorPropagation(t *testing.T) {
	t.Parallel()
	expected := errors.New("broken ID generator is broken")
	sh := shifted{
		gen:  broken{expected},
		bits: 1,
	}
	var tests = []struct {
		gen Interface
		err error
	}{
		{NewOverflowChecker(10, broken{expected}), expected},
		{sh, expected},
	}
	for _, test := range tests {
		for i := int64(1); i < 5; i++ {
			if _, err := test.gen.NewIDs(i); err.Error() != test.err.Error() {
				t.Errorf(
					"TestErrorPropagation %T/%v %d: got error %q, expected error %q",
					test.gen, test.gen, i, err, test.err)
			}
		}
	}
}

func TestSnowFlakeErrorPropagation(t *testing.T) {
	t.Parallel()
	gen := NewSnowflake(1 << 12)
	expected := fmt.Errorf("%T.NewIDs() overflow 1000000000000",
		gen.(*snowflake).constant.(shifted).gen.(overflowChecker).gen)
	if _, err := gen.NewIDs(1); !matchErrors(err, expected) {
		t.Errorf("TestSnowFlakeErrorPropagation: got error %q, expected error %q",
			err, expected)
	}
	gen = NewSnowflake(0)
	var tests = []struct {
		error
		int64
	}{
		{nil, 1 << 12},
		{
			fmt.Errorf("%T.NewIDs() overflow 1000000000000",
				gen.(*snowflake).seqChecker.(overflowChecker).gen),
			1,
		},
	}
	for _, test := range tests {
		if _, err := gen.NewIDs(test.int64); !matchErrors(err, test.error) {
			t.Errorf("TestSnowFlakeErrorPropagation: got error %q, expected error %q",
				err, test.error)
		}
	}
}

func matchErrors(a, b error) bool {
	var s1, s2 string
	if a != nil {
		s1 = a.Error()
	}
	if b != nil {
		s2 = b.Error()
	}
	return s1 == s2
}

func TestSnowflake(t *testing.T) {
	gen := NewSnowflake(3)
	var nodeMask int64 = (3 << 12)
	var counter int64
	for i := int64(0); i < 10; i++ {
		if i%2 == 0 {
			// Pairs 0 and 1, 2 and 3 ... produce generate under the same Millisecond,
			// provided the machine is fast enough. One possible improvement is to
			// calculate a NanoSecond sleep so clock is at the start of the Millisecond.
			time.Sleep(1 * time.Millisecond)
			counter = 0
		}
		counter += i
		v, err := gen.NewIDs(i + 1)
		tstamp := time.Now().UnixNano() / int64(time.Millisecond)
		expected := tstamp<<22 | nodeMask | counter
		switch {
		case err != nil:
			t.Errorf("TestSnowflake %d: expected %v, got error %q", i, expected, err)
		case math.Abs(float64(v-expected)) > 1.:
			t.Errorf("TestSnowflake %d: expected %v, got %v", i, expected, v)
		default:
			// success
		}
	}
}

type repeat struct{}

func (r repeat) NewIDs(count int64) (int64, error) {
	return count, nil
}

type broken struct {
	error
}

func (b broken) NewIDs(count int64) (int64, error) {
	return 0, b.error
}
