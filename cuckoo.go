// Copyright 2014 - Utkan Güngördü
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package cuckoo implements bucketized cuckoo hashing (also known as splash hashing).
// This implementation uses 4 hash functions and 8 cells per bucket.
// Greedy algorithm for collision resolution is a random walk.
package cuckoo

import (
	"math/rand"
	"runtime"
)

const (
	bshift         = 3 // Number of items in a bucket is 1<<bshift.
	blen           = 1 << bshift
	bmask          = blen - 1
	nhashshift     = 2
	nhash          = 1 << nhashshift // Number of hash functions. Make sure to change Cuckoo.hash and Cuckoo.shuffle accordingly when you change this.
	nhashmask      = nhash - 1
	invalidIndex   = -1
	invalidHash    = 0xffffffff
	gc             = true        // triegger GC after every alloc (which happens during grow)
	maxLogSize     = 32 - bshift // assuming 32-bit keys
	DefaultLogSize = 8           // Initial number of the buckets is 1<<DefaultLogSize
)

type Key uint32      // Must be an integer-type.
type Value uint32 // Can be anything, replace this to match your needs (not using unsafe.Pointer to avoid the overhead to store additional pointer or interface{} which comes with a worse overhead).
type Hash uint32

type bucket struct {
	keys [blen]Key
	vals [blen]Value
}

// Cuckoo implements a space-efficient map[Key]Value equivalent where Key is an integer and Value can be anything.
// Similar to built-in maps Cuckoo is not thread-safe. In a parallel environment you may need to wrap access with mutexes.
type Cuckoo struct {
	logsize  int // len(buckets) is 1<<logsize
	buckets  []bucket
	nentries int
	ngrow    int
	// To avoid allocating a bitmap for bucket usage, we use the default value of key (which is 0) to indicate that the entry is not used.
	// Instead of forbidding items with key==0 (and exposing an implementation quirk to the user), we use an additional
	// information: a slot is unsed iff buckets[i].keys[j]==0 AND zeroindex is not i<<bshift + j.
	// Thus, zeroindex is the index of the element with 0 key (invalidIndex means no 0 key) (throughout the package "index" means: lower blen bits indicate bucket index, upper bits indicate the bucket number.)
	zeroindex int
	// evacuated leftover item
	eitem bool
	ekey  Key
	eval  Value
	seed  [nhash]Hash // seed for hash functions
}

// Create a new cuckoo hash table with 2^logsize number of buckets initially.
// A single bucket can hold blen key/value pairs.
func NewCuckoo(logsize int) *Cuckoo {
	c := &Cuckoo{
		buckets:   make([]bucket, 1<<uint(logsize), 1<<uint(logsize)),
		logsize:   logsize,
		zeroindex: invalidIndex,
	}

	for i := range c.seed {
		c.seed[i] = Hash(rand.Uint32())
	}

	return c
}

func (c *Cuckoo) Len() int {
	return c.nentries
}

func (c *Cuckoo) hash(key Key) (h [nhash]Hash) {
	mask := Hash((1 << uint(c.logsize)) - 1)

	if useaesenc {
		aeshash32_4(key, mask, &c.seed, &h)
		return
	}

	k := Hash(key)
	h[0] = k & mask
	h[1] = Hash(murmur3(uint32(k), uint32(c.seed[1]))) & mask
	h[2] = Hash(xx(uint32(k), uint32(c.seed[2]))) & mask
	h[3] = Hash(mem(uint32(k), uint32(c.seed[3]))) & mask

	return
}

func (c *Cuckoo) shuffle(h *[nhash]Hash) {
	// Fisher-Yates shuffle
	r := rand.Uint32()
	i := int(r & 3)
	h[3], h[i] = h[i], h[3]
	i = int((r >> 2) % 3)
	h[2], h[i] = h[i], h[2]
	i = int((r >> 4) & 2)
	h[1], h[i] = h[i], h[1]
}

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

	h := c.hash(k)
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

func (c *Cuckoo) Delete(k Key) {
	if k == 0 {
		c.zeroindex = invalidIndex
		return
	}

	h := c.hash(k)
	for _, hash := range &h {
		b := &c.buckets[int(hash)]
		for i, key := range &b.keys {
			if k == key {
				b.keys[i] = 0
				return
			}
		}
	}

	// TODO(utkan): shrink?

	return
}

func (c *Cuckoo) Insert(k Key, v Value) {
	for {
		if c.tryInsert(k, v) {
			return
		}

		for i := 1; ; i++ {
			if ok := c.tryGrow(i); ok {
				break
			}
		}
	}
}

func (c *Cuckoo) tryInsert(k Key, v Value) (inserted bool) {
	// check loadFactor and can grow if necessary?
	h := c.hash(k)

	// Are we just updating the value for an existing key?
	updated, availableIndex := c.tryUpdate(k, v, h)
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
	if c.tryGreedyAdd(k, v, h) {
		c.nentries++
		return true
	}

	// All failed.
	return false
}

// If we already have an element with the the key k, we just update the value.
// Otherwise, index of an available slot --if exists at all-- is returned.
func (c *Cuckoo) tryUpdate(k Key, v Value, h [nhash]Hash) (updated bool, availableIndex int) {
	availableIndex = invalidIndex
	zeroindex := c.zeroindex
	
	// TODO(utkan): SSE2/AVX2 version

	for _, hash := range &h {
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
func (c *Cuckoo) tryAdd(k Key, v Value, h [nhash]Hash, except Hash) (added bool) {
	zeroindex := c.zeroindex

	for _, hash := range &h {
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
func (c *Cuckoo) tryGreedyAdd(k Key, v Value, h [nhash]Hash) (added bool) {
	// Expected maximum number of steps is O(log(n)):
	// Frieze, Alan, Páll Melsted, and Michael Mitzenmacher. "An analysis of random-walk cuckoo hashing." SIAM Journal on Computing 40.2 (2011): 291-308.
	max := (1 + c.logsize)

	for step := 0; step < max; step++ {
		c.shuffle(&h) //
		r := rand.Uint32()
		// randomly choose the item to evict
		i := int(r & bmask)
		d := int((r >> bshift) & nhashmask)
		hash := h[d]
		b := &c.buckets[int(hash)]
		ekey, eval := b.keys[i], b.vals[i]
		b.keys[i], b.vals[i] = k, v
		// try to put the evicted item back
		ehash := c.hash(ekey)
		if c.tryAdd(ekey, eval, ehash, hash) {
			return true
		}

		// we're back to where we started, except with a new item.
		k = ekey
		v = eval
		h = ehash
	}

	c.ekey = k
	c.eval = v
	c.eitem = true
	return false
}

func (c *Cuckoo) LoadFactor() float64 {
	return float64(c.nentries) / float64(len(c.buckets)<<bshift)
}

func (c *Cuckoo) tryGrow(del int) (ok bool) {
	c.ngrow++

	oldBuckets := c.buckets
	oldZeroindex := c.zeroindex

	c.logsize += del
	if c.logsize > maxLogSize {
		panic("cuckoo: cannot grow any furher")
	}
	c.buckets = make([]bucket, 1<<uint(c.logsize), 1<<uint(c.logsize))
	c.zeroindex = invalidIndex
	
	// rehash everything; we get better load factors at the expense of CPU time.

	defer func() {
		if !ok {
			c.logsize -= del
			c.buckets = oldBuckets
			c.zeroindex = oldZeroindex
		} else {
			oldBuckets = nil
		}

		if gc {
			runtime.GC()
		}
	}()

	for bi := range oldBuckets {
		b := &oldBuckets[bi]
		bucket0 := bi << bshift
		for i, k := range &b.keys {
			if k == 0 && oldZeroindex != bucket0+i {
				continue
			}

			v := b.vals[i]
			h := c.hash(k)

			if c.tryAdd(k, v, h, invalidHash) {
				continue
			}

			if ok = c.tryGreedyAdd(k, v, h); !ok {
				return
			}
		}
	}

	if c.eitem {
		if ok = c.tryInsert(c.ekey, c.eval); !ok {
			return
		}
		c.eitem = false
	}

	ok = true
	return
}
