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

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// parseRange creates a sorted and uniqued []int by parsing arguments like:
// 12,34-56,1023,80
// So if given '1,5-9,6,120,11' it will return: [ 1 5 6 7 8 9 11 120 ]
func parseRange(arg string) ([]int, error) {
	args := strings.Split(arg, ",")
	var nums []int

	single := regexp.MustCompile("^[0-9]+$")
	space := regexp.MustCompile("^[0-9]+-[0-9]+$")

	errs := make([]error, 2)
	var i, b1, b2 int
	for _, v := range args {
		if ok := single.MatchString(v); ok { // single number
			i, errs[0] = strconv.Atoi(v)
			nums = append(nums, i)
		} else if ok = space.MatchString(v); ok { // range numbers
			bounds := strings.Split(v, "-")
			b1, errs[0] = strconv.Atoi(bounds[0])
			b2, errs[1] = strconv.Atoi(bounds[1])
			switch {
			case b1 < b2:
			case b1 == b2:
				nums = append(nums, b1)
				continue
			case b1 > b2:
				b1, b2 = b2, b1
			}
			for i := b1; i <= b2; i++ {
				nums = append(nums, i)
			}
		} else { // bad arguments
			errs[0] = errors.New("bad numeric argument: " + v)
		}

		for _, v := range errs {
			if v != nil {
				return []int{}, v
			}
		}
	}
	sort.Ints(nums)
	seen := make(map[int]bool)
	var out []int
	for _, v := range nums {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out, nil
}
