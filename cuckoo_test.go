package cuckoo

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

const bytes = 3

// basic test
func TestCuckooFilter(t *testing.T) {
	// new cuckoo filter
	cf := NewCuckooFilter(3)

	data := []byte("abc")
	// insert an element
	err := cf.Insert(data)
	assert.Nil(t, err)

	// delete an element
	err = cf.Delete(data)
	assert.Nil(t, err)

	// reinsert the same element
	err = cf.Insert("abc")
	assert.Nil(t, err)

	// search all possible locations of an element
	res, err := cf.SearchAll(data)
	assert.Nil(t, err)
	fmt.Println(res)
}

func TestScaleData(t *testing.T) {
	size := 1000
	dataset := generateBytes(size)
	cf := NewCuckooFilter(size)
	// test insert
	for i := 0; i < len(dataset); i++ {
		err := cf.Insert(dataset[i])
		assert.Nil(t, err)
	}
	// test delete
	for i := 0; i < len(dataset); i++ {
		err := cf.Delete(dataset[i])
		assert.Nil(t, err)
	}
}

func generateBytes(size int) [][]byte {
	res := make([][]byte, size)
	for i := 0; i < size; i++ {
		res[i] = make([]byte, bytes)
		rand.Read(res[i])
	}
	return res
}
