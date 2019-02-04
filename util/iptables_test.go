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

func TestValidatePorts(t *testing.T) {
	type args struct {
		ports []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "standard", args: args{ports: []string{"1", "65535"}}, wantErr: false},
		{name: "reserved", args: args{ports: []string{"0"}}, wantErr: true},
		{name: "negative", args: args{ports: []string{"-1"}}, wantErr: true},
		{name: "out of range", args: args{ports: []string{"65536"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePorts(tt.args.ports); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePorts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNetworks(t *testing.T) {
	type args struct {
		nets []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "localhost", args: args{nets: []string{"127.0.0.1"}}, wantErr: false},
		{name: "single add", args: args{nets: []string{"192.168.1.1"}}, wantErr: false},
		{name: "cidr", args: args{nets: []string{"192.168.1.0/24"}}, wantErr: false},
		{name: "too long", args: args{nets: []string{"192.168.1.1.1"}}, wantErr: true},
		{name: "octet out of range", args: args{nets: []string{"192.168.1.260"}}, wantErr: true},
		{name: "mask out of range", args: args{nets: []string{"192.168.1.0/33"}}, wantErr: true},
		{name: "invalid chars", args: args{nets: []string{"192.168.1.abc"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateNetworks(tt.args.nets); (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetworks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetRestrictedPortsFromTablesData(t *testing.T) {
	type args struct {
		data  []byte
		chain string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{name: "nil", args: args{data: nil, chain: "KUBE-RESTRICT-IP"}, want: nil},
		{name: "empty", args: args{data: []byte{}, chain: "KUBE-RESTRICT-IP"}, want: nil},
		{name: "no matches", args: args{data: []byte("The quick brown fox jumps over the lazy dog"), chain: "KUBE-RESTRICT-IP"},
			want: nil},
		{name: "single port - default chain name", args: args{data: []byte("-A INPUT -p tcp -m multiport --dports 80 -m comment --comment kube-restrict-ip -j KUBE-RESTRICT-IP"), chain: "KUBE-RESTRICT-IP"},
			want: []string{"80"}},
		{name: "two ports - default chain name", args: args{data: []byte("-A INPUT -p tcp -m multiport --dports 80,8080 -m comment --comment kube-restrict-ip -j KUBE-RESTRICT-IP"), chain: "KUBE-RESTRICT-IP"},
			want: []string{"80", "8080"}},
		{name: "single port - custom chain name", args: args{data: []byte("-A INPUT -p tcp -m multiport --dports 80 -m comment --comment kube-restrict-ip -j KUBE-RESTRICT-IP-1"), chain: "KUBE-RESTRICT-IP-1"},
			want: []string{"80"}},
		{name: "two ports - custom chain name", args: args{data: []byte("-A INPUT -p tcp -m multiport --dports 80,8080 -m comment --comment kube-restrict-ip -j KUBE-RESTRICT-IP-1"), chain: "KUBE-RESTRICT-IP-1"},
			want: []string{"80", "8080"}},
		{name: "not matched chain name", args: args{data: []byte("-A INPUT -p tcp -m multiport --dports 80,8080 -m comment --comment kube-restrict-ip -j KUBE-RESTRICT-IP-1"), chain: "KUBE-RESTRICT-IP"},
			want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRestrictedPortsFromTablesData(tt.args.data, tt.args.chain); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRestrictedPortsFromTablesData() = %v, want %v", got, tt.want)
			}
		})
	}
}
