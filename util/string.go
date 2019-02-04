// Copyright © 2019 Victor Antonovich <victor@antonovich.me>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"bytes"
	"reflect"
	"strings"
)

// Convert string slice/array to set-like map
func ToSet(a []string) map[string]bool {
	s := map[string]bool{}

	for _, e := range a {
		s[e] = true
	}

	return s
}

// Checks two slices/arrays are matched (have equal sets of unique elements)
func Matched(a1, a2 []string) bool {
	return reflect.DeepEqual(ToSet(a1), ToSet(a2))
}

// Compare two sets, old and new, and get added and deleted elements
func DiffSets(old map[string]bool, new map[string]bool) (added map[string]bool, deleted map[string]bool) {
	added = map[string]bool{}
	deleted = map[string]bool{}

	for k, _ := range new {
		if !old[k] {
			added[k] = true
		}
	}

	for k, _ := range old {
		if !new[k] {
			deleted[k] = true
		}
	}

	return added, deleted
}

// Join all words to line with space as delimiter
func JoinWords(words ...string) string {
	return strings.Join(words, " ")
}

// Write newline terminated line to the buffer
func WriteLine(lines *bytes.Buffer, line string) {
	lines.WriteString(line + "\n")
}
