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
	"reflect"
	"testing"
)

func TestDiffSets(t *testing.T) {
	type args struct {
		old map[string]bool
		new map[string]bool
	}
	tests := []struct {
		name        string
		args        args
		wantAdded   map[string]bool
		wantDeleted map[string]bool
	}{
		{name: "empty", args: args{new: map[string]bool{}, old: map[string]bool{}},
			wantAdded: map[string]bool{}, wantDeleted: map[string]bool{}},
		{name: "added only", args: args{new: map[string]bool{"1": true, "2": true}, old: map[string]bool{"1": true}},
			wantAdded: map[string]bool{"2": true}, wantDeleted: map[string]bool{}},
		{name: "deleted only", args: args{new: map[string]bool{}, old: map[string]bool{"1": true}},
			wantAdded: map[string]bool{}, wantDeleted: map[string]bool{"1": true}},
		{name: "added and deleted", args: args{new: map[string]bool{"2": true}, old: map[string]bool{"1": true}},
			wantAdded: map[string]bool{"2": true}, wantDeleted: map[string]bool{"1": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotDeleted := DiffSets(tt.args.old, tt.args.new)
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("DiffSets() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !reflect.DeepEqual(gotDeleted, tt.wantDeleted) {
				t.Errorf("DiffSets() gotDeleted = %v, want %v", gotDeleted, tt.wantDeleted)
			}
		})
	}
}
