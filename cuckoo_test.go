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

package cuckoo

import (
	"math"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
)

var n = int(2e6) // close enough to a power of 2, to test whether the LoadFactor is close to 1 or not.

var (
	gkeys   []Key
	gvals   []Value
	gmap    map[Key]Value
	logsize = int(math.Ceil(math.Log2(float64(n))))
)

var (
	mapBytes    uint64
	cuckooBytes uint64
)

var (
	mbench map[Key]Value
	cbench *Cuckoo
)

func readAlloc() uint64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.Alloc
}

func mkmap(n int) (map[Key]Value, []Key, []Value, uint64) {

	keys := make([]Key, n, n)
	vals := make([]Value, n, n)

	var v Value
	runtime.GC()
	before := readAlloc()

	m := make(map[Key]Value)
	for i := 0; i < n; i++ {
		k := Key(rand.Uint32())
		m[k] = v
		keys[i] = k
		vals[i] = v
	}

	after := readAlloc()

	return m, keys, vals, after - before
}

func init() {
	gmap, gkeys, gvals, mapBytes = mkmap(n)
}

func TestZero(t *testing.T) {
	c := NewCuckoo(logsize)
	var v Value

	for i := 0; i < 10; i++ {
		c.Insert(0, v)
		_, ok := c.Search(0)
		if !ok {
			t.Error("search failed")
		}
	}
}

func TestSimple(t *testing.T) {
	c := NewCuckoo(DefaultLogSize)
	for k, v := range gmap {
		c.Insert(k, v)
	}

	for k, v := range gmap {
		cv, ok := c.Search(k)
		if !ok {
			t.Error("not ok:", k, v, cv)
			return
		}
		if reflect.DeepEqual(cv, v) == false {
			t.Error("got: ", cv, " expected: ", v)
			return
		}
	}

	if c.Len() != len(gmap) {
		t.Error("got: ", c.Len(), " expected: ", len(gmap))
		return
	}

	ndeleted := 0
	maxdelete := len(gmap) * 95 / 100
	for k := range gmap {
		if ndeleted >= maxdelete {
			break
		}

		c.Delete(k)
		if v, ok := c.Search(k); ok == true {
			t.Error("got: ", v)
			return
		}

		ndeleted++

		if c.Len() != len(gmap)-ndeleted {
			t.Error("got: ", c.Len(), " expected: ", len(gmap)-ndeleted)
			return
		}
	}
}

func TestMem(t *testing.T) {
	runtime.GC()
	before := readAlloc()

	c := NewCuckoo(logsize)
	for k, v := range gmap {
		c.Insert(k, v)
	}

	after := readAlloc()

	cuckooBytes = after - before

	t.Log("LoadFactor:", c.LoadFactor())
	t.Log("Built-in map memory usage (MiB):", float64(mapBytes)/float64(1<<20))
	t.Log("Cuckoo hash  memory usage (MiB):", float64(cuckooBytes)/float64(1<<20))
}

func BenchmarkCuckooInsert(b *testing.B) {
	cbench = NewCuckoo(logsize)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cbench.Insert(gkeys[i%n], gvals[i%n])
	}
}

func BenchmarkCuckooSearch(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cbench.Search(gkeys[i%n])
	}
}

func BenchmarkCuckooDelete(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cbench.Delete(gkeys[i%n])
	}
}

func BenchmarkMapInsert(b *testing.B) {
	mbench = make(map[Key]Value)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mbench[gkeys[i%n]] = gvals[i%n]
	}
}

func BenchmarkMapSearch(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mbench[gkeys[i%n]]
	}
}

func BenchmarkMapDelete(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		delete(mbench, gkeys[i%n])
	}
}
