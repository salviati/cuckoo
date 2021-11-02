package cuckoo

import (
	"math/big"
	"reflect"
	"unsafe"
)

// deleteElement deletes the element with given location from bucket
func deleteElement(val *bucket, data interface{}) {
	if reflect.DeepEqual(val.node, data) {
		val.node = nil
		return
	}
	for it := val.stash.Front(); it != val.stash.Back(); it = it.Next() {
		if it.Value == data {
			val.stash.Remove(it)
			return
		}
	}
}

// stashAppend appends a data to a bucket
func stashAppend(val *bucket, data interface{}, cf *CuckooFilter) bool {
	val.stash.PushBack(data)
	cf.cycleCount = 0
	return true
}

// convert data to uint32, if data is not a num, string or []byte will return error
func convert(data interface{}) (uint32, bool) {
	switch reflect.TypeOf(data).Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return *(*uint32)(unsafe.Pointer(&data)), true
	case reflect.String, reflect.Slice:
		var d []byte
		if s, ok := data.(string); ok {
			d = []byte(s)
		} else {
			d, ok = data.([]byte)
			if !ok {
				return 0, false
			}
		}
		key := uint32(new(big.Int).SetBytes(d).Uint64())
		return key, true
	}
	return 0, false
}
