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
	"bytes"
	"github.com/3cky/kube-restrict-ip/util"
	"github.com/golang/glog"
	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"
	"time"
)

type AppConfig struct {
	ConfigCheckInterval time.Duration
	IpChainName         string
	RestrictedPorts     []string
	AllowedNetworks     []string
}

func NewAppConfig(chainName string, ports, nets []string) *AppConfig {
	return &AppConfig{
		IpChainName:     chainName,
		RestrictedPorts: ports,
		AllowedNetworks: nets,
	}
}

type App struct {
	cfg      *AppConfig
	iptables utiliptables.Interface
}

func NewApp(cfg *AppConfig) *App {
	execer := utilexec.New()
	dbus := utildbus.New()
	protocol := utiliptables.ProtocolIpv4
	iptables := utiliptables.New(execer, dbus, protocol)
	return &App{
		cfg:      cfg,
		iptables: iptables,
	}
}

func (app *App) RunOnce() {
	// Fetch running config
	oldCfg := app.fetchRunningConfigFromTables()

	if err := app.updateTables(oldCfg, app.cfg); err != nil {
		glog.Fatalf("can't update iptables: %v", err)
	}

	glog.V(2).Info("iptables rules updated")
}

func (app *App) Run(cfgCh chan *AppConfig, doneCh chan struct{}) {
	defer close(doneCh)

	glog.Info("starting")

	// Do initial iptables rules synchronization
	glog.Info("do initial iptables rules sync")

	oldCfg := app.fetchRunningConfigFromTables()

	if err := app.updateTables(oldCfg, app.cfg); err != nil {
		glog.Errorf("initial iptables rules sync error: %v", err)
	} else {
		glog.Info("initial iptables rules sync done")
	}

	for {
		newCfg, ok := <-cfgCh

		if !ok {
			break
		}

		// Update iptables according to the updated config
		if err := app.updateTables(app.cfg, newCfg); err != nil {
			glog.Errorf("iptables rules sync error: %v", err)
		} else {
			glog.Info("iptables rules sync done")
			app.cfg = newCfg
		}
	}

	glog.Info("stopped")
}

func (app *App) updateTables(oldCfg, newCfg *AppConfig) error {
	// Create rules in iptables-restore format
	d := app.createTablesRestoreData(oldCfg, newCfg)
	glog.V(4).Infof("iptables-restore data:\n%s", d)

	// Update iptables rules
	err := app.iptables.RestoreAll(d, utiliptables.NoFlushTables, utiliptables.NoRestoreCounters)
	if err != nil {
		return err
	}

	return nil
}

// Fetch running config from iptables rules, if present
func (app *App) fetchRunningConfigFromTables() *AppConfig {
	var cfg *AppConfig = nil

	d := bytes.NewBuffer(nil)

	err := app.iptables.SaveInto(utiliptables.TableFilter, d)

	if err == nil {
		ports := util.GetRestrictedPortsFromTablesData(d.Bytes(), app.cfg.IpChainName)

		if ports != nil {
			cfg = NewAppConfig(app.cfg.IpChainName, ports, nil)
		}
	} else {
		glog.Errorf("can't fetch running config from iptables: %v", err)
	}

	return cfg
}

// Create iptables-restore data for synchronizing old config to new one
func (app *App) createTablesRestoreData(oldCfg, newCfg *AppConfig) []byte {
	lines := bytes.NewBuffer(nil)

	// Begin with table name ('filter')
	util.WriteLine(lines, "*"+string(utiliptables.TableFilter))

	// Add migration rules, if needed
	if oldCfg != nil {
		if oldCfg.IpChainName != newCfg.IpChainName {
			// Flush old network rules chain
			util.WriteLine(lines, util.CreateEmptyChainRule(oldCfg.IpChainName))
		}
		// Create/flush new network rules chain
		util.WriteLine(lines, util.CreateEmptyChainRule(newCfg.IpChainName))

		// Check INPUT rule for redirecting restricted ports to network rules chain should be updated
		if (oldCfg.IpChainName != newCfg.IpChainName) || !util.Matched(oldCfg.RestrictedPorts, newCfg.RestrictedPorts) {
			// Delete old rule
			util.WriteLine(lines, util.CreateRestrictedPortsDeleteRule(oldCfg.IpChainName, oldCfg.RestrictedPorts))
			// Add new rule
			util.WriteLine(lines, util.CreateRestrictedPortsAddRule(newCfg.IpChainName, newCfg.RestrictedPorts))
		}
	} else {
		// Flush network rules chain
		util.WriteLine(lines, util.CreateEmptyChainRule(newCfg.IpChainName))
		// Add INPUT rule for redirecting restricted ports to network rules chain
		util.WriteLine(lines, util.CreateRestrictedPortsAddRule(newCfg.IpChainName, newCfg.RestrictedPorts))
	}

	// Write rules for all allowed networks to the chain
	for _, net := range newCfg.AllowedNetworks {
		util.WriteLine(lines, util.CreateAllowedNetworkChainRule(newCfg.IpChainName, net))
	}

	// Write default (DROP) rule for unmatched networks at the end of network rules chain
	util.WriteLine(lines, util.CreateDefaultNetworkChainRule(newCfg.IpChainName))

	// Commit all rules
	util.WriteLine(lines, "COMMIT")

	return lines.Bytes()
}
