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

const (
	blen      = 1 << bshift
	bmask     = blen - 1
	nhash     = 1 << nhashshift
	nhashmask = nhash - 1
)

type bucket struct {
	keys [blen]Key
	vals [blen]Value
}

// Cuckoo implements a memory-efficient map[Key]Value equivalent where Key is an integer and Value can be anything.
// Similar to built-in maps Cuckoo is not thread-safe. In a parallel environment you may need to wrap access with mutexes.
type Cuckoo struct {
	logsize  int // len(buckets) is 1<<logsize.
	buckets  []bucket
	nentries int
	ngrow    int
	nshrink  int
	nrehash  int
	// To avoid allocating a bitmap for bucket usage, we use the default value of key (which is 0) to indicate that the entry is not used.
	// Instead of forbidding items with key==0 (and exposing an implementation quirk to the user), we use zeroValue and zeroIsSet to store
	// an item with 0 key. Hence, there is no key/value with key==0 within buckets and any bucket with key==0 is empty.
	zeroValue Value // Value of the item with Key==0 is placed here.
	zeroIsSet bool  // true if there is an item with Key==0.
	// evacuated leftover item.
	eitem bool
	ekey  Key
	eval  Value
	seed  [nhash]hash // seed for hash functions.
}

var zero Value

func alloc(n int) []bucket {
	return make([]bucket, n, n)
}

func init() {
	// ensures the sanity of the config
	if nhash*nhashshift+bshift+nhashshift > 63 {
		panic("cuckoo: tryGreedyAdd needs nhash*nhashshift + bshift + nhashshift bits of random data; either modify tryGreedyAdd or reduce nhash/bshift.")
	}

	if nhashshift > 8 {
		panic("cuckoo: nhashshift is too large. either modify Cuckoo.shuffle or reduce nhashshift.")
	}
}

// NewCuckoo creates a new cuckoo hash table with 2^logsize number of key/value cells initially.
//
// If you can estimate the number of unique items n (unique here refers to keys, not values) you are going to insert,
// choosing a proper logsize [which is math.Ceil(math.Log2(n))] here is strongly advised.
// Doing so will avoid grows, which are computationally expensive and require allocation.
func NewCuckoo(logsize int) *Cuckoo {
	logsize -= bshift

	if logsize <= 0 {
		logsize = 1
	}

	if logsize > hashBits {
		panic("cuckoo: log size is too")
	}

	c := &Cuckoo{
		buckets: alloc(1 << uint(logsize)),
		logsize: logsize,
	}

	c.reseed()

	return c
}

func (c *Cuckoo) reseed() {
	for i := range &c.seed {
		c.seed[i] = hash(rand.Uint32())
	}
}

// Len returns the number of items in the hash map.
func (c *Cuckoo) Len() int {
	return c.nentries
}

// default hash function
func defaultHash(k Key, seed hash) hash {
	return hash(xx_32(uint32(k), uint32(seed)))
}

func (c *Cuckoo) dohash(key Key, h *[nhash]hash) {
	mask := hash((1 << uint(c.logsize)) - 1)

	for i := range h {
		h[i] = defaultHash(key, c.seed[i]) & mask
	}

	return
}

// uses lowest nhash*nhashshift bits of r.
func (c *Cuckoo) shuffle(h *[nhash]hash, r int64) {
	// Fisher-Yates shuffle
	for j := nhash - 1; j > 0; j-- {
		i := int(uint8(r&nhashmask) % uint8(j+1)) // we assume nhashshift <= 8 here (some archs lack div instruction).
		h[i], h[j] = h[j], h[i]
		r >>= nhashshift
	}
}

// Search tries to retrieve the value associated with the given key.
// If no such item is found, ok is set to false.
func (c *Cuckoo) Search(k Key) (v Value, ok bool) {
	if k == 0 {
		if c.zeroIsSet == false {
			return
		}

		return c.zeroValue, true
	}

	// TODO(utkan): SSE2/AVX2 version

	var h [nhash]hash
	c.dohash(k, &h)
	for _, hval := range &h {
		b := &c.buckets[int(hval)]
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
		c.zeroIsSet = false
		c.zeroValue = zero
		return
	}

	var h [nhash]hash
	c.dohash(k, &h)
	for _, hval := range &h {
		b := &c.buckets[int(hval)]
		for i, key := range &b.keys {
			if k == key {
				c.nentries--
				b.keys[i] = 0
				b.vals[i] = zero
				break
			}
		}
	}

	if 1<<uint(c.logsize+bshift-shrinkFactor) > c.nentries {
		// TODO(utkan): depending on the current load factorm starting from shrinkFactor-1 may be better.
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
	if k == 0 {
		c.zeroIsSet = true
		c.zeroValue = v
		return
	}

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
	var h [nhash]hash
	c.dohash(k, &h)

	// Are we just updating the value for an existing key?
	updated, freeSlot, ibucket, index := c.tryUpdate(k, v, &h)
	if updated {
		return true
	}

	// Nope, do we have an empty slot?
	if freeSlot {
		c.addAt(k, v, ibucket, index)
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
func (c *Cuckoo) tryUpdate(k Key, v Value, h *[nhash]hash) (updated bool, freeSlot bool, ibucket int, index int) {
	// TODO(utkan): SSE2/AVX2 version

	for _, bi := range h {
		b := &c.buckets[int(bi)]

		for i, key := range &b.keys {
			if k == key {
				b.vals[i] = v
				updated = true
				return
			}

			if freeSlot == false && key == 0 {
				ibucket = int(bi)
				index = i
				freeSlot = true
			}
		}
	}
	return
}

func (c *Cuckoo) addAt(k Key, v Value, ibucket int, index int) {
	b := &c.buckets[ibucket]
	b.keys[index] = k
	b.vals[index] = v
}

// Used by tryGrow and tryGreedyAdd.
// Similar to tryUpdate, but tryAdd assumes there is no item with key already.
// tryAdd also omits the slot given by the parameter except, when ignore is set to true.
func (c *Cuckoo) tryAdd(k Key, v Value, h *[nhash]hash, ignore bool, except hash) (added bool) {
	if k == 0 {
		c.zeroIsSet = true
		c.zeroValue = v
		return
	}

	for _, hval := range h {
		if ignore && except == hval {
			continue
		}

		bi := int(hval)
		b := &c.buckets[bi]

		for i, key := range &b.keys {
			if key == 0 {
				b.keys[i] = k
				b.vals[i] = v

				return true
			}
		}
	}
	return false
}

// tryUpdate and tryAdd both failed. Let's try moving the eggs around.
// This implementation uses random walk.
func (c *Cuckoo) tryGreedyAdd(k Key, v Value, h *[nhash]hash) (added bool) {
	// Expected maximum number of steps is O(log(n)):
	// Frieze, Alan, Páll Melsted, and Michael Mitzenmacher. "An analysis of random-walk cuckoo hashing." SIAM Journal on Computing 40.2 (2011): 291-308.
	max := (1 + c.logsize) * randomWalkCoefficient

	var ehash [nhash]hash

	for step := 0; step < max; step++ {
		r := rand.Int63() // need nhash*nhashshift + bshift + nhashshift random bits
		c.shuffle(h, r)
		r >>= nhash * nhashshift
		// randomly choose the item to evict
		i := int(r & bmask)
		d := int((r >> bshift) & nhashmask)
		hval := h[d]
		b := &c.buckets[int(hval)]
		ekey, eval := b.keys[i], b.vals[i]
		b.keys[i], b.vals[i] = k, v
		// try to put the evicted item back
		c.dohash(ekey, &ehash)
		if c.tryAdd(ekey, eval, &ehash, true, hval) {
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
		if cnew.logsize <= 8 {
			return
		}
		cnew.nshrink++
	}

	cnew.logsize += δ
	if cnew.logsize > hashBits {
		panic("cuckoo: cannot grow any furher")
	}
	cnew.buckets = alloc(1 << uint(cnew.logsize))

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

	var h [nhash]hash

	for bi := range c.buckets {
		b := c.buckets[bi]
		for i, k := range &b.keys {
			if k == 0 {
				continue
			}

			v := b.vals[i]
			cnew.dohash(k, &h)

			if cnew.tryAdd(k, v, &h, false, 0) {
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
