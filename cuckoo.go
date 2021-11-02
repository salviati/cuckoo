// Copyright (c) 2014-2015 Utkan Güngördü <utkan@freeconsole.org>
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
	"container/list"
	"fmt"
	"math"
	"math/rand"
)

type bucket struct {
	node  interface{}
	stash *list.List
}

type CuckooFilter struct {
	Filter     map[uint32]*bucket
	Seed       uint32
	cycleCount byte
	hasher     [3]hashFunc
}

// NewCuckooFilter creates a new cuckoo filter, if seed == 0, then will generate a new seed
// if you want to use your own hash function, you can pass it, otherwise it will use default functions
func NewCuckooFilter(dataSize int, seed uint32, hasher [3]hashFunc) *CuckooFilter {
	cuckoo := &CuckooFilter{
		Filter: make(map[uint32]*bucket, int(math.Ceil(float64(dataSize)*growParameter))),
	}
	if seed != 0 {
		cuckoo.Seed = seed
	}
	if hasher[0] == nil {
		hasher[0] = murmur3_32
	}
	if hasher[1] == nil {
		hasher[1] = xx_32
	}
	if hasher[2] == nil {
		hasher[2] = mem_32
	}
	cuckoo.hasher = hasher
	return cuckoo
}

// ReSeed the rand generator, if seed == 0, then generate a random number instead
func (cf *CuckooFilter) ReSeed(seed uint32) {
	if seed != 0 {
		cf.Seed = seed
		return
	}
	cf.Seed = rand.Uint32()
}

// Insert data into Filter
func (cf *CuckooFilter) Insert(data interface{}) bool {
	cf.cycleCount += 1
	key, ok := convert(data)
	if !ok {
		return false
	}
	if _, ok := cf.insert(data, key, cf.hasher[0]); ok {
		return true
	}
	if _, ok := cf.insert(data, key, cf.hasher[1]); ok {
		return true
	}
	if val, ok := cf.insert(data, key, cf.hasher[2]); ok {
		return true
	} else if cf.cycleCount < 3 {
		kicked := val.node
		deleteElement(val, kicked)
		val.node = data
		return cf.Insert(kicked)
	}
	return false
}

// Delete data from Filter
func (cf *CuckooFilter) Delete(data interface{}) bool {
	key, ok := convert(data)
	if !ok {
		return false
	}
	if ok := cf.delete(data, key, murmur3_32); ok {
		return true
	} else if ok := cf.delete(data, key, xx_32); ok {
		return true
	} else {
		return cf.delete(data, key, mem_32)
	}
}

// SearchAll possible buckets given a certain data
func (cf *CuckooFilter) SearchAll(data interface{}) ([]uint32, bool) {
	key, ok := convert(data)
	if !ok {
		return nil, false
	}
	return []uint32{murmur3_32(key, cf.Seed), xx_32(key, cf.Seed), mem_32(key, cf.Seed)}, true
}

func (cf *CuckooFilter) insert(data interface{}, key uint32, hasher hashFunc) (*bucket, bool) {
	try := hasher(key, cf.Seed)
	if val, ok := cf.Filter[try]; !ok {
		cf.Filter[try] = &bucket{
			node:  data,
			stash: list.New(),
		}
		cf.cycleCount = 0
		return val, true
	} else if val.node == nil {
		val.node = data
		cf.cycleCount = 0
		return val, true
	} else if cf.cycleCount == 3 {
		return val, stashAppend(val, data, cf)
	}
	return cf.Filter[try], false
}

func (cf *CuckooFilter) delete(data interface{}, key uint32, hasher hashFunc) bool {
	input := hasher(key, cf.Seed)
	if val, ok := cf.Filter[input]; ok {
		deleteElement(val, data)
		return true
	}
	return false
}

// fmt format
func (cf *CuckooFilter) String() string {
	if len(cf.Filter) == 0 {
		return "nil filter"
	}
	output := ""
	for k, v := range cf.Filter {
		output += fmt.Sprintf("--------[%v]--------\nnode=[%v]\n", k, v.node)
		output += "stash = ["
		for i := v.stash.Front(); i != nil; i = i.Next() {
			output += fmt.Sprintf("%v ", i.Value)
		}
		output += "]\n"
	}
	return output
}
