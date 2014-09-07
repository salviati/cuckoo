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

package cuckoo

import (
	"time"
)

type fastrand struct {
	x uint32
}

func newFastrand() *fastrand {
	return &fastrand{x: 0x49f6428a + uint32(time.Now().UnixNano())}
}

// fastrand implementation from runtime package
func (r *fastrand) next() uint32 {
	x := r.x
	x ^= (((x << 1) >> 31) & 0x88888eef) ^ 1
	r.x = x
	return x
}
