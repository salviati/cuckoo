# cuckoo
<img src="http://upload.wikimedia.org/wikipedia/commons/thumb/1/16/NeomorphusSalviniSmit.jpg/220px-NeomorphusSalviniSmit.jpg"></img>

Package `cuckoo` implements d-ary bucketized [cuckoo hashing](http://en.wikipedia.org/wiki/Cuckoo_hashing) with stash (bucketized cuckoo hashing is also known as splash tables).
This implementation uses configurable number of hash functions and cells per bucket.
Greedy algorithm for collision resolution is a random walk.

## Purpose
Cuckoo is a memory-efficient alternative to the built-in `map[Key]Value` type (where Key is an integer type and Value can be any type) with zero per-item overhead.
Hence, the memory-efficiency is equal to the [load factor](http://en.wikipedia.org/wiki/Hash_table#Key_statistics) (which can be as high as 99%).

## Performance
Benchmark results on linux/amd64 with i7-4770S (2M uint32 Key/Value insertion) with gc compiler (version 1.5):

	go test -bench=. -v -cpu 1
	=== RUN   TestZero
	--- PASS: TestZero (0.00s)
	=== RUN   TestSimple
	--- PASS: TestSimple (1.62s)
	=== RUN   TestMem
	--- PASS: TestMem (0.48s)
			cuckoo_test.go:148: LoadFactor: 0.9534401893615723
			cuckoo_test.go:149: Built-in map memory usage (MiB): 65.56954193115234
			cuckoo_test.go:150: Cuckoo hash  memory usage (MiB): 16.000137329101562
	PASS
	BenchmarkCuckooInsert   10000000               134 ns/op               0 B/op          0 allocs/op
	BenchmarkCuckooSearch   20000000                95.1 ns/op             0 B/op          0 allocs/op
	BenchmarkCuckooDelete   20000000               149 ns/op               0 B/op          0 allocs/op
	BenchmarkMapInsert      10000000               151 ns/op               9 B/op          0 allocs/op
	BenchmarkMapSearch      20000000                91.3 ns/op             0 B/op          0 allocs/op
	BenchmarkMapDelete      100000000               11.8 ns/op             0 B/op          0 allocs/op
	ok      github.com/salviati/cuckoo      14.734s

### Arch-specific optimizations
Cuckoo is [SIMD-friendly](http://www1.cs.columbia.edu/~kar/pubsk/icde2007.pdf) by design, and there are plans for fast-path SIMD functions.

## Usage
After cloning the repository, modify the definitions of `Key` and `Value` types to fit your needs. For optimal performance, you should also experiment with the fine-grade parameters of the algorithm listed in `config.go`.

## Documentation
[godoc](http://godoc.org/github.com/salviati/cuckoo)

## License
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see [http://www.gnu.org/licenses/](http://www.gnu.org/licenses/).
