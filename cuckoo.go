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
	"errors"
	"math"
	"math/rand"
)

type bucket [stashSize]interface{}

type CuckooFilter struct {
	Filter     map[uint32]*bucket
	Seed       uint32
	cycleCount byte
}

func NewCuckooFilter(dataSize int) *CuckooFilter {
	return &CuckooFilter{
		Filter: make(map[uint32]*bucket, int(math.Ceil(float64(dataSize)*growParameter))),
		Seed:   rand.Uint32(),
	}
}

// ReSeed the rand generator
func (cf *CuckooFilter) ReSeed(seed uint32) {
	cf.Seed = seed
}

// Insert data into Filter
func (cf *CuckooFilter) Insert(data interface{}) error {
	cf.cycleCount += 1
	key, err := convert(data)
	if err != nil {
		return err
	}
	if _, ok := cf.insert(data, key, murmur3_32); ok {
		return nil
	}
	if _, ok := cf.insert(data, key, xx_32); ok {
		return nil
	}
	if val, ok := cf.insert(data, key, mem_32); ok {
		return nil
	} else if cf.cycleCount < 3 {
		kicked := val[0]
		if err := deleteElement(val, 0); err != nil {
			return err
		}
		val[0] = data
		return cf.Insert(kicked)
	}
	return errors.New("need or more hash more stash")
}

// Delete data from Filter
func (cf *CuckooFilter) Delete(data interface{}) error {
	key, err := convert(data)
	if err != nil {
		return err
	}
	if err := cf.delete(data, key, murmur3_32); err == nil {
		return nil
	} else if err := cf.delete(data, key, xx_32); err == nil {
		return nil
	} else {
		return cf.delete(data, key, mem_32)
	}
}

// SearchAll possible buckets given a certain data
func (cf *CuckooFilter) SearchAll(data interface{}) ([]uint32, error) {
	key, err := convert(data)
	if err != nil {
		return nil, err
	}
	return []uint32{murmur3_32(key, cf.Seed), xx_32(key, cf.Seed), mem_32(key, cf.Seed)}, nil
}

func (cf *CuckooFilter) insert(data interface{}, key uint32, hasher hashFunc) (*bucket, bool) {
	try := hasher(key, cf.Seed)
	if val, ok := cf.Filter[try]; !ok {
		cf.Filter[try] = &bucket{data}
		cf.cycleCount = 0
		return val, true
	} else if val[0] == nil || (cf.cycleCount == 3 && val[stashSize-1] == nil) {
		if err := arrayAppend(val, data, cf); err == nil {
			return val, true
		} else {
			return nil, false
		}
	}
	return &bucket{}, false
}

func (cf *CuckooFilter) delete(data interface{}, key uint32, hasher hashFunc) error {
	input := hasher(key, cf.Seed)
	if val, ok := cf.Filter[input]; ok {
		loc := search(val, data)
		if err := deleteElement(val, loc); err != nil {
			return err
		}
		return nil
	}
	return errors.New("cannot delete element for some unknown reason")
}
