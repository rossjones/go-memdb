package memdb

import (
	"reflect"
	"testing"

	"golang.org/x/exp/slices"
)

func TestSimpleWAL(t *testing.T) {

	type TestStruct struct {
		ID    string
		Value string
	}

	type TestCase struct {
		ID     string
		Table  string
		Object *TestStruct
		Delete bool
	}

	testcases := []TestCase{
		{"0001", "test", &TestStruct{ID: "0001", Value: "Data1"}, false},
		{"0002", "test", &TestStruct{ID: "0002", Value: "Data2"}, false},
		{"0003", "test", &TestStruct{ID: "0003", Value: "Data3"}, false},
		{"0002", "test", &TestStruct{ID: "0002", Value: "Data2"}, true}, // deletion
	}

	schema := &DBSchema{
		Tables: map[string]*TableSchema{
			"test": { // &TableSchema
				Name: "test",
				Indexes: map[string]*IndexSchema{
					"id": { // &IndexSchema
						Name:    "id",
						Unique:  true,
						Indexer: &StringFieldIndex{Field: "ID"},
					},
				},
				Type: reflect.TypeOf((*TestStruct)(nil)).Elem(),
			},
		},
	}

	tempDirectory := t.TempDir()

	db, err := NewMemDB(schema, tempDirectory)
	if err != nil {
		t.FailNow()
	}

	for _, testcase := range testcases {
		tx := db.Txn(true)
		if testcase.Delete {
			tx.Delete(testcase.Table, testcase.Object)
		} else {
			tx.Insert(testcase.Table, testcase.Object)
		}

		tx.Commit()
	}

	countItems := 0
	collectsIDs := make([]string, 0, 2)

	tx := db.Txn(false)
	iter, _ := tx.Get("test", "id")
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		ts := obj.(*TestStruct)

		countItems += 1
		collectsIDs = append(collectsIDs, ts.ID)
	}
	tx.Commit()

	// We deleted 2 so it should not appear in the new database
	if slices.Compare(collectsIDs, []string{"0001", "0003"}) != 0 {
		t.FailNow()
	}

	db = nil

	// Create a new database and rely on replay to bring us back to the
	// same state

	dbr, err := NewMemDB(schema, tempDirectory)
	if err != nil {
		t.FailNow()
	}

	countItems = 0
	collectsIDs = make([]string, 0, 2)

	tx = dbr.Txn(false)
	iter, _ = tx.Get("test", "id")
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		ts := obj.(TestStruct)

		countItems += 1
		collectsIDs = append(collectsIDs, ts.ID)
	}
	tx.Commit()

	// We deleted 2 so it should not appear in the new database
	if slices.Compare(collectsIDs, []string{"0001", "0003"}) != 0 {
		t.FailNow()
	}
}
