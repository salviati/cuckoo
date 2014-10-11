// Copyright (c) 2014 Utkan Güngördü <utkan@freeconsole.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package cuckoo implements d-ary bucketized cuckoo hashing (bucketized cuckoo hashing is also known as splash tables).
// This implementation uses configurable number of hash functions and cells per bucket.
// Greedy algorithm for collision resolution is a random walk.
//
// This implementation prioritizes memory-efficiency over speed.
package cuckoo

import (
	"math/rand"
	"runtime"
)

// configurable variables (for tuning the algorithm)
const (
	bshift                = 3   // Number of items in a bucket is 1<<bshift.
	nhashshift            = 3   // Number of hash functions is 1<<nhashshift.
	shrinkFactor          = 2   // A shrink will be triggered when the load factor goes below 2^(-shrinkFactor).
	rehashThreshold       = 0.9 // If the load factor is below rehashThreshold, Insert will try to rehash everything before actually growing.
	randomWalkCoefficient = 1   // A multiplicative coefficient best determined by benchmarks. The optimal value depends on bshift and nhashshift.
)

// other configurable variables
const (
	gc             = false // trigger GC after every alloc (which happens during grow).
	DefaultLogSize = 8     // Initial number of the buckets is 1<<DefaultLogSize.
)

const (
	maxLogSize   = 32 - bshift // (# of bit in Hash type) - bshift. Default assumes 32-bit keys.
	blen         = 1 << bshift
	bmask        = blen - 1
	nhash        = 1 << nhashshift
	nhashmask    = nhash - 1
	invalidIndex = -1
	invalidHash  = 0xffffffff
)

// Key must be an integer-type.
type Key uint32

// Value can be anything, replace this to match your needs (not using unsafe.Pointer to avoid the overhead to store additional pointer or interface{} which comes with a worse overhead).
type Value uint32

type bucket struct {
	keys [blen]Key
	vals [blen]Value
}

// Cuckoo implements a space-efficient map[Key]Value equivalent where Key is an integer and Value can be anything.
// Similar to built-in maps Cuckoo is not thread-safe. In a parallel environment you may need to wrap access with mutexes.
type Cuckoo struct {
	logsize  int // len(buckets) is 1<<logsize.
	buckets  []bucket
	nentries int
	ngrow    int
	nshrink  int
	nrehash  int
	// To avoid allocating a bitmap for bucket usage, we use the default value of key (which is 0) to indicate that the entry is not used.
	// Instead of forbidding items with key==0 (and exposing an implementation quirk to the user), we use an additional
	// information: a slot is unsed iff buckets[i].keys[j]==0 AND zeroindex is not i<<bshift + j.
	// Thus, zeroindex is the index of the element with 0 key (invalidIndex means no 0 key) (throughout the package "index" means: lower blen bits indicate bucket index, upper bits indicate the bucket number).
	zeroindex int
	// evacuated leftover item.
	eitem bool
	ekey  Key
	eval  Value
	seed  [nhash]Hash // seed for hash functions.
}

func alloc(n int) []bucket {
	return make([]bucket, n, n)
}

func init() {
	if nhash*nhashshift+bshift+nhashshift > 63 {
		panic("tryGreedyAdd needs nhash*nhashshift + bshift + nhashshift bits of random data; either modify tryGreedyAdd or reduce nhash/bshift.")
	}
}

// NewCuckoo creates a new cuckoo hash table with 2^logsize number of buckets initially.
// A single bucket can hold blen key/value pairs.
func NewCuckoo(logsize int) *Cuckoo {

	c := &Cuckoo{
		buckets:   alloc(1 << uint(logsize)),
		logsize:   logsize,
		zeroindex: invalidIndex,
	}

	c.reseed()

	return c
}

func (c *Cuckoo) reseed() {
	for i := range &c.seed {
		c.seed[i] = Hash(rand.Uint32())
	}
}

// Len returns the number of items in the hash map.
func (c *Cuckoo) Len() int {
	return c.nentries
}

// default hash function
func defaultHash(k Key, seed Hash) Hash {
	return Hash(xx(uint32(k), uint32(seed)))
}

func (c *Cuckoo) hash(key Key, h *[nhash]Hash) {
	mask := Hash((1 << uint(c.logsize)) - 1)

	for i := range h {
		h[i] = defaultHash(key, c.seed[i]) & mask
	}

	return
}

// uses lowest nhash*nhashshift bits of r.
func (c *Cuckoo) shuffle(h *[nhash]Hash, r int64) {
	// Fisher-Yates shuffle
	for j := nhash - 1; j > 0; j-- {
		i := int(r&nhashmask) % (j + 1)
		h[i], h[j] = h[j], h[i]
		r >>= nhashshift
	}
}

// Search tries to retrieve the value associated with the given key.
// If no such item is found, ok is set to false.
func (c *Cuckoo) Search(k Key) (v Value, ok bool) {
	if k == 0 {
		if c.zeroindex == invalidIndex {
			return
		}

		bi := c.zeroindex >> bshift
		i := c.zeroindex & bmask
		return c.buckets[bi].vals[i], true
	}

	// TODO(utkan): SSE2/AVX2 version

	var h [nhash]Hash
	c.hash(k, &h)
	for _, hash := range &h {
		b := &c.buckets[int(hash)]
		for i, key := range &b.keys {
			if k == key {
				return b.vals[i], true
			}
		}
	}
	return
}

// Delete removes the item corresponding to the given key (if exists).
func (c *Cuckoo) Delete(k Key) {
	if k == 0 {
		c.zeroindex = invalidIndex
		return
	}

	var h [nhash]Hash
	c.hash(k, &h)
	for _, hash := range &h {
		b := &c.buckets[int(hash)]
		for i, key := range &b.keys {
			if k == key {
				c.nentries--
				b.keys[i] = 0
				return
			}
		}
	}

	if 1<<uint(c.logsize+bshift-shrinkFactor) > c.nentries {
		for i := shrinkFactor; i > 0; i-- {
			if c.tryGrow(-i) {
				break
			}
		}
	}

	return
}

// Insert adds given key/value item into the hash map.
// If an item with key k already exists, it will be replaced.
func (c *Cuckoo) Insert(k Key, v Value) {
	for {
		if c.tryInsert(k, v) {
			return
		}

		i0 := 1
		if c.LoadFactor() < rehashThreshold {
			i0 = 0
		}

		for i := i0; ; i++ {
			if ok := c.tryGrow(i); ok {
				break
			}
		}
	}
}

func (c *Cuckoo) tryInsert(k Key, v Value) (inserted bool) {
	// check loadFactor and can grow if necessary?
	var h [nhash]Hash
	c.hash(k, &h)

	// Are we just updating the value for an existing key?
	updated, availableIndex := c.tryUpdate(k, v, &h)
	if updated {
		return true
	}

	// Nope, do we have an empty slot?
	if availableIndex != invalidIndex {
		c.addAt(k, v, availableIndex)
		c.nentries++
		return true
	}

	// Nope again, lets try moving the eggs around.
	if c.tryGreedyAdd(k, v, &h) {
		c.nentries++
		return true
	}

	// All failed.
	return false
}

// If we already have an element with the the key k, we just update the value.
// Otherwise, index of an available slot --if exists at all-- is returned.
func (c *Cuckoo) tryUpdate(k Key, v Value, h *[nhash]Hash) (updated bool, availableIndex int) {
	availableIndex = invalidIndex
	zeroindex := c.zeroindex

	// TODO(utkan): SSE2/AVX2 version

	for _, hash := range h {
		bi := int(hash)
		b := &c.buckets[bi]
		bucket0 := bi << bshift

		for i, key := range &b.keys {
			if k == key {
				if k == 0 {
					c.zeroindex = (bi << bshift) + i
				}
				b.vals[i] = v
				updated = true
				return
			}

			if availableIndex == invalidIndex && key == 0 && (zeroindex == invalidIndex || zeroindex != bucket0+i) {
				availableIndex = bucket0 + i
			}
		}
	}
	return
}

func (c *Cuckoo) addAt(k Key, v Value, index int) {
	b := &c.buckets[index>>bshift]
	i := index & bmask
	b.keys[i] = k
	b.vals[i] = v
}

// We did tryUpdate, and it turned out that there is no element with key k.
// Now, see if there's an empty slot we can add key-value into.
// The array h is accessed in random order.
func (c *Cuckoo) tryAdd(k Key, v Value, h *[nhash]Hash, except Hash) (added bool) {
	zeroindex := c.zeroindex

	for _, hash := range h {
		if except != invalidHash && except == hash {
			continue
		}

		bi := int(hash)
		b := &c.buckets[bi]
		bucket0 := bi << bshift

		for i, key := range &b.keys {
			if key == 0 && (zeroindex == invalidIndex || zeroindex != bucket0+i) { // is this an empty slot? zeroindex == invalidIndex may help with branch prediction.
				b.keys[i] = k
				b.vals[i] = v

				if k == 0 {
					c.zeroindex = bucket0 + i
				}

				return true
			}
		}
	}
	return false
}

// tryUpdate and tryAdd both failed. Let's try moving the eggs around.
// This implementation uses random walk.
func (c *Cuckoo) tryGreedyAdd(k Key, v Value, h *[nhash]Hash) (added bool) {
	// Expected maximum number of steps is O(log(n)):
	// Frieze, Alan, Páll Melsted, and Michael Mitzenmacher. "An analysis of random-walk cuckoo hashing." SIAM Journal on Computing 40.2 (2011): 291-308.
	max := (1 + c.logsize) * randomWalkCoefficient

	var ehash [nhash]Hash

	for step := 0; step < max; step++ {
		r := rand.Int63() // need nhash*nhashshift + bshift + nhashshift random bits
		c.shuffle(h, r)
		r >>= nhash * nhashshift
		// randomly choose the item to evict
		i := int(r & bmask)
		d := int((r >> bshift) & nhashmask)
		hash := h[d]
		b := &c.buckets[int(hash)]
		ekey, eval := b.keys[i], b.vals[i]
		b.keys[i], b.vals[i] = k, v
		// try to put the evicted item back
		c.hash(ekey, &ehash)
		if c.tryAdd(ekey, eval, &ehash, hash) {
			return true
		}

		// we're back to where we started, except with a new item.
		k = ekey
		v = eval
		*h = ehash
	}

	c.ekey = k
	c.eval = v
	c.eitem = true
	return false
}

// LoadFactor returns the load factor of the hash table, which is the
// ratio of the used cells to the allocated cells.
func (c *Cuckoo) LoadFactor() float64 {
	return float64(c.nentries) / float64(len(c.buckets)<<bshift)
}

// Tries to grow the hash table by a factor of 2^δ.
func (c *Cuckoo) tryGrow(δ int) (ok bool) {
	// NOTE(utkan): reads during grow are OK.
	cnew := &Cuckoo{}
	*cnew = *c
	cnew.reseed()

	if δ == 0 {
		cnew.nrehash++
	}

	if δ > 0 {
		cnew.ngrow++
	}

	if δ < 0 {
		cnew.nshrink++
	}

	cnew.logsize += δ
	if cnew.logsize > maxLogSize {
		panic("cuckoo: cannot grow any furher")
	}
	cnew.buckets = alloc(1 << uint(cnew.logsize))
	cnew.zeroindex = invalidIndex

	// rehash everything; we get better load factors at the expense of CPU time.

	defer func() {
		if ok {
			*c = *cnew
		}

		cnew = nil

		if gc {
			runtime.GC()
		}
	}()

	var h [nhash]Hash

	for bi := range c.buckets {
		b := c.buckets[bi]
		bucket0 := bi << bshift
		for i, k := range &b.keys {
			if k == 0 && c.zeroindex != bucket0+i {
				continue
			}

			v := b.vals[i]
			cnew.hash(k, &h)

			if cnew.tryAdd(k, v, &h, invalidHash) {
				continue
			}

			if ok = cnew.tryGreedyAdd(k, v, &h); !ok {
				return
			}
		}
	}

	if cnew.eitem {
		if ok = cnew.tryInsert(cnew.ekey, cnew.eval); !ok {
			return
		}
		cnew.eitem = false
	}

	ok = true
	return
}
