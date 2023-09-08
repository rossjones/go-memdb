package memdb

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"sync"

	"golang.org/x/exp/slices"
)

type PersistedObject struct {
	Table      string
	Before     interface{}
	After      interface{}
	PrimaryKey []byte
}

func (p *PersistedObject) From(chg Change) {
	p.Table = chg.Table
	p.Before = chg.Before
	p.After = chg.After
	p.PrimaryKey = chg.primaryKey
}

func (p *PersistedObject) ToChange() Change {
	return Change{
		Table:      p.Table,
		Before:     p.Before,
		After:      p.After,
		primaryKey: p.PrimaryKey,
	}
}

type WAL interface {
	WriteEntry(chg Change) error
	Replay(schema *DBSchema) chan Change
}

type SimpleWAL struct {
	folder  string
	entries int

	mu sync.Mutex
}

// NewSimpleWAL creates a simple (and unsafe) wal where entries are
// written to a folder in ever increasing numbers which will sort
// lexicographically when attempting to replay them.
func NewSimpleWAL(location string) (*SimpleWAL, error) {
	e, err := os.ReadDir(location)
	if err != nil {
		return nil, err
	}

	return &SimpleWAL{
		folder:  location,
		entries: len(e),
	}, nil
}

// Replay implements WAL.
func (s *SimpleWAL) Replay(schema *DBSchema) chan Change {
	ch := make(chan Change)

	go func() {
		entries, _ := os.ReadDir(s.folder)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			names = append(names, e.Name())
		}

		slices.Sort(names)

		for _, name := range names {
			var object PersistedObject

			data, _ := os.ReadFile(path.Join(s.folder, name))
			_ = json.Unmarshal(data, &object)

			typ := schema.Tables[object.Table].Type
			if object.Before != nil {
				object.Before = s.convertMapToStruct(object.Before.(map[string]interface{}), typ)
			}
			if object.After != nil {
				object.After = s.convertMapToStruct(object.After.(map[string]interface{}), typ)
			}

			ch <- object.ToChange()
		}

		close(ch)
	}()

	return ch
}

func (s *SimpleWAL) convertMapToStruct(m map[string]interface{}, structType reflect.Type) interface{} {
	structValue := reflect.New(structType).Elem()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)
		fieldName := fieldType.Name

		if val, ok := m[fieldName]; ok {
			field.Set(reflect.ValueOf(val))
		}
	}

	return structValue.Interface()
}

// WriteEntry implements WAL by writing a single file with the
// provided change. If the value of the object (`Change.After`)
// is nil, then this is a deletion.
func (s *SimpleWAL) WriteEntry(chg Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fname := fmt.Sprintf("%012d.log", s.entries)
	s.entries += 1

	pc := &PersistedObject{}
	pc.From(chg)

	target := path.Join(s.folder, fname)
	data, err := json.Marshal(pc)
	if err != nil {
		return err
	}

	return os.WriteFile(target, data, 0644)
}

var _ WAL = (*SimpleWAL)(nil)
