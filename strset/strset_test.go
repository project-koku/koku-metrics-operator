/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package strset

import (
	"testing"
)

func TestAdd(t *testing.T) {
	strSet := NewSet()
	strSet.Add("key1")
	strSet.Add("key2")
	strSet.Add("key3")
	if strSet.Len() != 3 {
		t.Errorf("expected strSet %v to have length 3 but got %d", strSet, strSet.Len())
	}
}

func TestContains(t *testing.T) {
	strSet := NewSet()
	strSet.Add("key1")
	if strSet.Contains("key1") == false {
		t.Errorf("expected strSet %v to contain key 'key1' but it does not", strSet)
	}
	if strSet.Contains("key2") == true {
		t.Errorf("expected strSet %v to not contain key 'key2' but it does", strSet)
	}
}

func TestRemove(t *testing.T) {
	strSet := NewSet()
	strSet.Add("key1")
	strSet.Add("key2")
	if strSet.Contains("key1") == false {
		t.Errorf("expected strSet %v to contain 'key1' but it does not", strSet)
	}
	if strSet.Contains("key2") == false {
		t.Errorf("expected strSet %v to contain 'key2' but it does not", strSet)
	}
	if strSet.Len() != 2 {
		t.Errorf("expected strSet %v to have length 2 but got %d", strSet, strSet.Len())
	}

	if strSet.Contains("key0") == true {
		t.Errorf("expected strSet %v not to contain 'key0' but it does", strSet)
	}
	strSet.Remove("key0")
	if strSet.Contains("key1") == false {
		t.Errorf("after removing 'key0' expected strSet %v to contain 'key1' but it does not", strSet)
	}
	if strSet.Contains("key2") == false {
		t.Errorf("after removing 'key0' expected strSet %v to contain 'key2' but it does not", strSet)
	}
	if strSet.Len() != 2 {
		t.Errorf("after removing 'key0' expected strSet %v to stay 2 but got %d", strSet, strSet.Len())
	}

	strSet.Remove("key1")
	if strSet.Contains("key1") == true {
		t.Errorf("after removing 'key1' expected strSet %v to not contain 'key1' but it does", strSet)
	}
	if strSet.Contains("key2") == false {
		t.Errorf("after removing 'key1' expected strSet %v to contain 'key2' but it does not", strSet)
	}
	if strSet.Len() != 1 {
		t.Errorf("after removing 'key1' expected strSet %v to have length 1 but got %d", strSet, strSet.Len())
	}
}

func TestRange(t *testing.T) {
	tcs := []struct {
		name string
		keys []string
	}{
		{name: "strset contains 3 items", keys: []string{"ashley", "michael", "boaz"}},
		{name: "empty strset", keys: []string{}},
		{name: "one item in strset", keys: []string{"gopher"}},
	}
	for _, tc := range tcs {
		strSet := NewSet()
		items := []string{}
		for _, key := range tc.keys {
			strSet.Add(key)
		}
		rangeChn := strSet.Range()
		for item := range rangeChn {
			items = append(items, item)
		}
		if len(items) != len(tc.keys) {
			t.Errorf("expected number of items %v (%d) to be equal to %v (%d)", items, len(items), tc.keys, len(tc.keys))
		}
	}
}
