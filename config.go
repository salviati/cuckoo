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

// configurable variables (for tuning the algorithm)
const (
	bshift                = 3   // Number of items in a bucket is 1<<bshift.
	nhashshift            = 2   // Number of hash functions is 1<<nhashshift. (With SSE2, we can do 4 at once).
	shrinkFactor          = 0   // A shrink will be triggered when the load factor goes below 2^(-shrinkFactor). Setting this to 0 will disable shrinking and avoid potential new allocations.
	rehashThreshold       = 0.9 // If the load factor is below rehashThreshold, Insert will try to rehash everything before actually growing.
	randomWalkCoefficient = 2   // A multiplicative coefficient best determined by benchmarks. The optimal value depends on bshift and nhashshift.
	stashSize             = 4   // Size of stash (see Kirsch, Adam, Michael Mitzenmacher, and Udi Wieder. "More robust hashing: Cuckoo hashing with a stash." SIAM Journal on Computing 39.4 (2009): 1543-1561.)
)

// other configurable variables
const (
	gc             = false      // trigger GC after every alloc (which happens during grow).
	DefaultLogSize = 8 + bshift // A reasonable logsize value for NewCuckoo for use when the number of items to be inserted is not known ahead.
)

// Key must be an integer-type.
type Key uint32

// Value can be anything, replace this to match your needs (not using unsafe.Pointer to avoid the overhead to store additional pointer or interface{} which comes with a worse overhead).
type Value uint32
