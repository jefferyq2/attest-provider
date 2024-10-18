/*
   Copyright Docker attest-provider authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"testing"
)

func Test_nameValuePairs_Set(t *testing.T) {
	tests := []struct {
		name    string
		nvp     nameValuePairs
		wantErr bool
		input   string
	}{
		{name: "no value", nvp: nameValuePairs{}, input: "key=value", wantErr: false},
		{name: "invalid", nvp: nameValuePairs{}, input: "keyvalue", wantErr: true},
		{name: "empty", nvp: nameValuePairs{}, input: "", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.nvp.Set(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("nameValuePairs.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
