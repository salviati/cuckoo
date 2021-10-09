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
	stashSize     = 4   // Size of stash (see Kirsch, Adam, Michael Mitzenmacher, and Udi Wieder. "More robust hashing: Cuckoo hashing with a stash." SIAM Journal on Computing 39.4 (2009): 1543-1561.)
	growParameter = 1.2 // the parameter determine how much more space we need to alloc based on the dataSize
)
