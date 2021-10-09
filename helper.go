package cuckoo

import (
	"errors"
	"math/big"
	"reflect"
	"unsafe"
)

// deleteElement deletes the element with given location from bucket
func deleteElement(val *bucket, loc int) error {
	if loc == -1 {
		return nil
	}
	if loc >= stashSize || loc < 0 {
		return errors.New("index out of range")
	}
	val[loc] = nil
	return nil
}

// arrayAppend appends a data to a bucket
func arrayAppend(val *bucket, data interface{}, cf *CuckooFilter) error {
	for i := 0; i < stashSize; i++ {
		if val[i] == nil {
			val[i] = data
			cf.cycleCount = 0
			return nil
		}
	}
	return errors.New("no spare space in stash")
}

// search the index of given data in the bucket, if more than one element have the same value, return the first one
func search(val *bucket, data interface{}) int {
	for i := 0; i < stashSize; i++ {
		if reflect.DeepEqual(val[i], data) {
			return i
		}
	}
	return -1
}

// convert data to uint32, if data is not string or []byte will return error
func convert(data interface{}) (uint32, error) {
	switch reflect.TypeOf(data).Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return *(*uint32)(unsafe.Pointer(&data)), nil
	}
	var d []byte
	if s, ok := data.(string); ok {
		d = []byte(s)
	} else {
		d, ok = data.([]byte)
		if !ok {
			return 0, errors.New("cannot convert data to byte slice")
		}
	}
	key := uint32(new(big.Int).SetBytes(d).Uint64())
	return key, nil
}
