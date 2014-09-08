// Copyright (c) 2014 Utkan Güngördü <utkan@freeconsole.org>
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

package cuckoo

import (
	"reflect"
	"unsafe"
)

var bucketSize = int(unsafe.Sizeof(bucket{}))

func byteToBucketSlice(bytes []byte) (buckets []bucket) {
	bytesh := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	bucketsh := (*reflect.SliceHeader)(unsafe.Pointer(&buckets))

	bucketsh.Data = bytesh.Data
	bucketsh.Len = bytesh.Len / bucketSize
	bucketsh.Cap = bytesh.Cap / bucketSize

	return
}

func allocBuckets(malloc func(size int) []byte, nbuckets int) []bucket {
	bytes := malloc(bucketSize * nbuckets)
	return byteToBucketSlice(bytes)
}
