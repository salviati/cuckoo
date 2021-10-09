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

type hashFunc func(k uint32, seed uint32) uint32

const (
	murmur3_c1_32 uint32 = 0xcc9e2d51
	murmur3_c2_32 uint32 = 0x1b873593
)

const (
	xx_prime32_1 uint32 = 2654435761
	xx_prime32_2 uint32 = 2246822519
	xx_prime32_3 uint32 = 3266489917
	xx_prime32_4 uint32 = 668265263
	xx_prime32_5 uint32 = 374761393
)

const (
	mem_c0 = 2860486313
	mem_c1 = 3267000013
)

func murmur3_32(k uint32, seed uint32) uint32 {
	k *= murmur3_c1_32
	k = (k << 15) | (k >> (32 - 15))
	k *= murmur3_c2_32

	h := seed
	h ^= k
	h = (h << 13) | (h >> (32 - 13))
	h = (h<<2 + h) + 0xe6546b64

	return h
}

func xx_32(k uint32, seed uint32) uint32 {
	h := seed + xx_prime32_5
	h += k * xx_prime32_3
	h = ((h << 17) | (h >> (32 - 17))) * xx_prime32_4
	h ^= h >> 15
	h *= xx_prime32_2
	h ^= h >> 13
	h *= xx_prime32_3
	h ^= h >> 16

	return h
}

func mem_32(k uint32, seed uint32) uint32 {
	h := k ^ mem_c0
	h ^= (k & 0xff) * mem_c1
	h ^= (k >> 8 & 0xff) * mem_c1
	h ^= (k >> 16 & 0xff) * mem_c1
	h ^= (k >> 24 & 0xff) * mem_c1

	return h
}
