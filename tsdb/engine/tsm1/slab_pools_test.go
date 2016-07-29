package tsm1

import "testing"

// TestByteSliceSlabPool is a sanity check, verifying reference counting (and
// thus re-use).
func TestByteSliceSlabPool(t *testing.T) {
	p := NewByteSliceSlabPool()
	checkRefs := func(n int64) {
		if p.refs != n {
			t.Fatalf("bad refs %d", n)
		}
	}
	checkRefs(0)
	empty := p.Get(0)
	checkRefs(0)
	if empty != nil {
		t.Fatal("bad bufA")
	}

	bufA := p.Get(1)
	checkRefs(1)

	p.Inc(bufA)
	checkRefs(2)

	bufA[0] = 'x'

	bufB := p.Get(1)
	checkRefs(3)
	bufB[0] = 'y'

	if bufA[0] != 'x' {
		t.Fatal("bad overwrite")
	}

	p.Dec(bufA)
	checkRefs(2)
	p.Dec(bufA)
	checkRefs(1)

	bufC := p.Get(1)
	checkRefs(2)
	bufC[0] = 'z'

	if bufC[0] != 'z' {
		t.Fatal("bad reuse")
	}
}

func TestStringSlabPoolUnsharded(t *testing.T) {
	p := NewStringSlabPool(1)
	s0, b0 := p.Get(1, -1)
	b0[0] = 'x'
	if s0 != "x" {
		t.Fatal("bad string write s0")
	}

	s1, b1 := p.Get(1, -1)
	b1[0] = 'y'
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "x" {
		t.Fatal("bad overwrite s0")
	}

	p.Dec(s0)

	s2, b2 := p.Get(1, -1)
	b2[0] = 'z'
	if s2 != "z" {
		t.Fatal("bad string write s2")
	}
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "z" {
		t.Fatal("expected overwrite s0")
	}
}
func TestStringSlabPoolSharded1(t *testing.T) {
	p := NewStringSlabPool(16)
	shardID := p.SmartShardID()
	s0, b0 := p.Get(1, shardID)
	b0[0] = 'x'
	if s0 != "x" {
		t.Fatal("bad string write s0")
	}

	s1, b1 := p.Get(1, shardID)
	b1[0] = 'y'
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "x" {
		t.Fatal("bad overwrite s0")
	}

	p.Dec(s0)

	s2, b2 := p.Get(1, shardID)
	b2[0] = 'z'
	if s2 != "z" {
		t.Fatal("bad string write s2")
	}
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "z" {
		t.Fatal("expected overwritten s0")
	}
}
func TestStringSlabPoolSharded2(t *testing.T) {
	p := NewStringSlabPool(16)
	shardID := p.SmartShardID()
	s0, b0 := p.Get(1, shardID)
	b0[0] = 'x'
	if s0 != "x" {
		t.Fatal("bad string write s0")
	}

	s1, b1 := p.Get(1, shardID)
	b1[0] = 'y'
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "x" {
		t.Fatal("bad overwrite s0")
	}

	p.Dec(s0)

	shardID2 := p.SmartShardID()
	s2, b2 := p.Get(1, shardID2)
	b2[0] = 'z'
	if s2 != "z" {
		t.Fatal("bad string write s2")
	}
	if s1 != "y" {
		t.Fatal("bad string write s1")
	}
	if s0 != "x" {
		t.Fatal("expected preserved s0")
	}
}
func TestIntegerValueSlabPoolUnsharded(t *testing.T) {
	p := NewIntegerValueSlabPool(1)
	iv0 := p.Get(-1)
	iv0.unixnano = 1000
	iv0.value = 1001

	iv1 := p.Get(-1)
	iv1.unixnano = 2000
	iv1.value = 2001

	if iv0.unixnano != 1000 {
		t.Fatal("bad overwrite iv0.unixnano")
	}
	if iv0.value != 1001 {
		t.Fatal("bad overwrite iv0.value")
	}

	p.Dec(iv0)

	iv2 := p.Get(-1)
	iv2.unixnano = 3000
	iv2.value = 3001
	if iv0.unixnano != 3000 {
		t.Fatal("expected overwrite with iv2.unixnano")
	}
	if iv0.value != 3001 {
		t.Fatal("expected overwrite with iv2.value")
	}
	if iv1.unixnano != 2000 {
		t.Fatal("unexpected overwrite iv1.unixnano")
	}
	if iv1.value != 2001 {
		t.Fatal("unexpected overwrite iv1.value")
	}
}

func TestFloatValueSlabPoolUnsharded(t *testing.T) {
	p := NewFloatValueSlabPool(1)
	iv0 := p.Get(-1)
	iv0.unixnano = 1000
	iv0.value = 1001.

	iv1 := p.Get(-1)
	iv1.unixnano = 2000
	iv1.value = 2001.

	if iv0.unixnano != 1000 {
		t.Fatal("bad overwrite iv0.unixnano")
	}
	if iv0.value != 1001. {
		t.Fatal("bad overwrite iv0.value")
	}

	p.Dec(iv0)

	iv2 := p.Get(-1)
	iv2.unixnano = 3000
	iv2.value = 3001.
	if iv0.unixnano != 3000 {
		t.Fatal("expected overwrite with iv2.unixnano")
	}
	if iv0.value != 3001. {
		t.Fatal("expected overwrite with iv2.value")
	}
	if iv1.unixnano != 2000 {
		t.Fatal("unexpected overwrite iv1.unixnano")
	}
	if iv1.value != 2001. {
		t.Fatal("unexpected overwrite iv1.value")
	}
}
