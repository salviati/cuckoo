package cuckoo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_deleteElement(t *testing.T) {
	b := &bucket{byte(1), byte(2), byte(3), byte(4)}
	err := deleteElement(b, 0)
	assert.Nil(t, err)

	b = &bucket{byte(1), byte(2), byte(3), byte(4)}
	err = deleteElement(b, -1)
	assert.Nil(t, err)

	b = &bucket{byte(1), byte(2), byte(3), byte(4)}
	err = deleteElement(b, -2)
	assert.NotNil(t, err)
}

func Test_arrayAppend(t *testing.T) {
	cf := NewCuckooFilter(1)
	cf.Filter[0] = &bucket{[]byte{1}}
	err := arrayAppend(cf.Filter[0], []byte("a"), cf)
	assert.Nil(t, err)

	cf = NewCuckooFilter(1)
	cf.Filter[0] = &bucket{byte(1), byte(2), byte(3), byte(4)}
	err = arrayAppend(cf.Filter[0], []byte("a"), cf)
	assert.NotNil(t, err)
}

func Test_search(t *testing.T) {
	b := &bucket{[]byte{1}, []byte{2}, []byte{3}, []byte{4}}
	loc := search(b, []byte{1})
	assert.Equal(t, loc, 0)

	b = &bucket{[]byte{1}, []byte{2}, []byte{3}, []byte{4}}
	loc = search(b, []byte{10})
	assert.Equal(t, loc, -1)
}

func Test_convert(t *testing.T) {
	_, err := convert("a")
	assert.Nil(t, err)

	_, err = convert([]byte{97})
	assert.Nil(t, err)

	_, err = convert(15)
	assert.Nil(t, err)
}
