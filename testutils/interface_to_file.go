package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

/*

These funcs exist to generate the prometheus data for unit tests. In `getQueryResults` and `getQueryRangeResults`,
we would add

	testutils.Save(filepath.Join("test_files", "test_data", query.Name), {matrix|vector})

The operator must be running locally.

*/

// Marshal is a function that marshals the object into an
// io.Reader.
// By default, it uses the JSON marshaller.
var Marshal = func(v interface{}) (io.Reader, error) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// Save saves a representation of v to the file at path.
func Save(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	return err
}
