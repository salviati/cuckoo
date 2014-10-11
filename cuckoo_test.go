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
	"math/rand"
	"reflect"
	"testing"
)

var (
	gkeys   []Key
	gvals   []Value
	gmap    map[Key]Value
	n       = int(3e6)
	logSize = 19
)

func mkmap(n int) (map[Key]Value, []Key, []Value) {
	m := make(map[Key]Value)
	keys := make([]Key, n, n)
	vals := make([]Value, n, n)

	var v Value

	for i := 0; i < n; i++ {
		k := Key(rand.Uint32())
		m[k] = v
		keys[i] = k
		vals[i] = v
	}

	return m, keys, vals
}

func init() {
	gmap, gkeys, gvals = mkmap(n)
}

func TestZero(t *testing.T) {
	c := NewCuckoo(logSize)
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
	c := NewCuckoo(logSize)
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

	t.Log("LoadFactor:", c.LoadFactor())
}

func BenchmarkCuckooInsert(b *testing.B) {
	c := NewCuckoo(logSize)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Insert(gkeys[i%n], gvals[i%n])
	}
}

func BenchmarkCuckooSearch(b *testing.B) {
	c := NewCuckoo(logSize)
	for i := 0; i < len(gkeys); i++ {
		c.Insert(gkeys[i%n], gvals[i%n])
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c.Search(gkeys[i%n])
	}
}

func BenchmarkMapInsert(b *testing.B) {
	m := make(map[Key]Value)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		m[gkeys[i%n]] = gvals[i%n]
	}
}

func BenchmarkMapSearch(b *testing.B) {
	m := make(map[Key]Value)

	for i := 0; i < len(gkeys); i++ {
		m[gkeys[i%n]] = gvals[i%n]
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = m[gkeys[i%n]]
	}
}
