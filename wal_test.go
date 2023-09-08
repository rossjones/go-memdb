package memdb

import (
	"fmt"
	"testing"
)

func TestSimpleWAL(t *testing.T) {

	type TestStruct struct {
		Name string
	}

	ch := Change{
		Table:  "t",
		Before: TestStruct{Name: "Before"},
		After:  TestStruct{Name: "After"},
	}

	l := t.TempDir()
	w, _ := NewSimpleWAL(l)
	ch.primaryKey = []byte("0001")
	_ = w.WriteEntry(ch, false)
	ch.primaryKey = []byte("0002")
	_ = w.WriteEntry(ch, false)
	ch.primaryKey = []byte("0003")
	_ = w.WriteEntry(ch, false)

	c := w.Replay()

	for change := range c {
		fmt.Println(change.primaryKey)
	}
}
