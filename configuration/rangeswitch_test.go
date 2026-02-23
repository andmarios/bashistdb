// Copyright (c) 2015-2026, Marios Andreopoulos.
//
// This file is part of bashistdb.
//
//      Bashistdb is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
//      Bashistdb is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
//      You should have received a copy of the GNU General Public License
// along with bashistdb.  If not, see <http://www.gnu.org/licenses/>.

package configuration

import "testing"

func TestParseRange(t *testing.T) {

	const (
		_  = iota
		OK // We expect test to pass
		ER // We expect test to return error
	)

	test := []struct {
		want   []int
		expect int
		input  string
	}{
		{[]int{1, 2, 3, 4}, OK, "1,2,3,4,4"},
		{[]int{1, 3, 4, 5, 10, 11}, OK, "1,3-5,4,11,10"},
		{[]int{}, ER, "1,3-"},
		{[]int{}, ER, "1,3-"},
		{[]int{}, ER, "1,-3"},
		{[]int{}, ER, "1,s"},
		{[]int{}, ER, "1,"},
		{[]int{1, 2, 3, 4}, OK, "1,4-2"},
		{[]int{5, 9}, OK, "5-5,9"},
	}

	for _, v := range test {
		i, err := parseRange(v.input)

		switch v.expect {
		case OK:
			if err != nil {
				t.Fatal(err.Error())
			}
			if !compareIntSlice(i, v.want) {
				t.Fatalf("Got %v, wanted %v.\n", i, v.want)
			}
		case ER:
			if err == nil {
				t.Fatalf("Expected error. Got nil instead for input %v.\n", v.input)
			}
		}
	}
}

// Our slices are uniqued and sorted. This is why comparing is so simple.
func compareIntSlice(x, y []int) bool {
	if len(x) != len(y) {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
