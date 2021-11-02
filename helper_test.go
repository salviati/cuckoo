package cuckoo

import (
	"container/list"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newElement() *bucket {
	b := &bucket{
		node:  byte(1),
		stash: list.New(),
	}
	b.stash.PushBack(byte(2))
	b.stash.PushBack(byte(3))
	b.stash.PushBack(byte(4))
	return b
}

func Test_deleteElement(t *testing.T) {
	b := newElement()
	deleteElement(b, byte(2))

	b = newElement()
	deleteElement(b, byte(1))
	assert.Equal(t, nil, b.node)

	b = newElement()
	deleteElement(b, -1)
}

func Test_arrayAppend(t *testing.T) {
	cf := NewCuckooFilter(1, 0, [3]hashFunc{})
	cf.Filter[0] = &bucket{node: byte(1), stash: list.New()}
	ok := stashAppend(cf.Filter[0], []byte("a"), cf)
	assert.True(t, ok)

	cf = NewCuckooFilter(1, 0, [3]hashFunc{})
	cf.Filter[0] = newElement()
	cf.Filter[0].stash.PushBack(byte(5))
	ok = stashAppend(cf.Filter[0], []byte("a"), cf)
	assert.NotNil(t, ok)
}

func Test_convert(t *testing.T) {
	res1, ok := convert("a")
	assert.True(t, ok)
	fmt.Println(res1)

	res2, ok := convert([]byte{97})
	assert.True(t, ok)
	fmt.Println(res2)

	_, ok = convert(15)
	assert.True(t, ok)
}
