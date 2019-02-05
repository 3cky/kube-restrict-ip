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

package app

import (
	"github.com/3cky/kube-restrict-ip/util"
	"reflect"
	"testing"

	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	testiptables "k8s.io/kubernetes/pkg/util/iptables/testing"
)

func TestApp_fetchRunningConfigFromTables(t *testing.T) {
	type fields struct {
		cfg      *AppConfig
		iptables utiliptables.Interface
	}
	tests := []struct {
		name   string
		fields fields
		want   *AppConfig
	}{
		{
			name: "empty data",
			fields: struct {
				cfg      *AppConfig
				iptables utiliptables.Interface
			}{cfg: NewAppConfig("TEST-CHAIN", nil, nil),
				iptables: testiptables.NewFake()},
			want: nil,
		},
		{
			name: "non-matching chain",
			fields: struct {
				cfg      *AppConfig
				iptables utiliptables.Interface
			}{cfg: NewAppConfig("TEST-CHAIN", nil, nil),
				iptables: &testiptables.FakeIPTables{Lines: []byte(util.JoinWords("-A", "INPUT",
					util.CreateRestrictedPortsMatchRule("TEST-CHAIN-1", []string{"1234", "3456"})))}},
			want: nil,
		},
		{
			name: "matching chain",
			fields: struct {
				cfg      *AppConfig
				iptables utiliptables.Interface
			}{cfg: NewAppConfig("TEST-CHAIN", []string{}, []string{}),
				iptables: &testiptables.FakeIPTables{Lines: []byte(util.JoinWords("-A", "INPUT",
					util.CreateRestrictedPortsMatchRule("TEST-CHAIN", []string{"1234", "3456"})))}},
			want: NewAppConfig("TEST-CHAIN", []string{"1234", "3456"}, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				cfg:      tt.fields.cfg,
				iptables: tt.fields.iptables,
			}
			if got := app.fetchRunningConfigFromTables(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("App.fetchRunningConfigFromTables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApp_updateTables(t *testing.T) {
	type fields struct {
		cfg      *AppConfig
		iptables utiliptables.Interface
	}
	type args struct {
		oldCfg *AppConfig
		newCfg *AppConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "chain name updated",
			fields: struct {
				cfg      *AppConfig
				iptables utiliptables.Interface
			}{
				cfg:      NewAppConfig("", nil, nil),
				iptables: testiptables.NewFake(),
			},
			args: struct {
				oldCfg *AppConfig
				newCfg *AppConfig
			}{
				oldCfg: NewAppConfig("TEST-CHAIN", []string{"1234"}, []string{}),
				newCfg: NewAppConfig("TEST-CHAIN-NEW", []string{"4567"}, []string{"127.0.0.1"})},
			want: `*filter
:TEST-CHAIN - [0:0]
:TEST-CHAIN-NEW - [0:0]
-D INPUT -p tcp -m multiport --dports 1234 -m comment --comment "kube-restrict-ip" -j TEST-CHAIN
-I INPUT 1 -p tcp -m multiport --dports 4567 -m comment --comment "kube-restrict-ip" -j TEST-CHAIN-NEW
-A TEST-CHAIN-NEW -s 127.0.0.1 -j RETURN
-A TEST-CHAIN-NEW -j REJECT --reject-with icmp-port-unreachable
COMMIT
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				cfg:      tt.fields.cfg,
				iptables: tt.fields.iptables,
			}
			err := app.updateTables(tt.args.oldCfg, tt.args.newCfg)
			if err != nil {
				t.Errorf("App.updateTables() error = %v", err)
			}
			got := app.iptables.(*testiptables.FakeIPTables).Lines
			if got == nil || tt.want != string(got) {
				t.Errorf("App.updateTables() Lines '%s', want '%s'", got, tt.want)
			}
		})
	}
}
