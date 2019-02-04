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

package cmd

import (
	"errors"
	"flag"
	"fmt"
	"github.com/3cky/kube-restrict-ip/util"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/3cky/kube-restrict-ip/app"
	"github.com/3cky/kube-restrict-ip/log"
	"github.com/3cky/kube-restrict-ip/pkg/build"
)

const (
	FlagRunOnce             = "once"
	FlagVersion             = "version"
	FlagConfigCheckInterval = "check-interval"
	FlagIpChainName         = "ip-chain"
	FlagRestrictedPorts     = "restricted-ports"
	FlagAllowedNetworks     = "allowed-networks"
	FlagConfigFileName      = "config-file"

	ConfigCheckInterval   = "checkInterval"
	ConfigIpChainName     = "ipChain"
	ConfigRestrictedPorts = "restrictedPorts"
	ConfigAllowedNetworks = "allowedNetworks"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "kube-restrict-ip",
		Long: "Restrict Kubernetes ports access by IP using iptables.",
		Run:  runCmd,
	}
	initCmd(cmd)
	return cmd
}

func initCmd(cmd *cobra.Command) {
	// Command-related flags set
	f := cmd.Flags()
	f.BoolP(FlagVersion, "V", false, "display the build number and timestamp")
	f.Bool(FlagRunOnce, false, "run once and exit")
	f.StringP(FlagConfigFileName, "c", "",
		fmt.Sprintf("config file name to watch (implied '%s' if omitted)", FlagRunOnce))
	f.DurationP(FlagConfigCheckInterval, "t", 10*time.Second, "config file update check interval")
	f.String(FlagIpChainName, "KUBE-RESTRICT-IP", "iptables chain name")
	f.StringSlice(FlagRestrictedPorts, nil, "restricted ports")
	f.StringSlice(FlagAllowedNetworks, nil, "allowed networks")

	// Merge flags
	pflag.CommandLine.SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if strings.Contains(name, "_") {
			return pflag.NormalizedName(strings.Replace(name, "_", "-", -1))
		}
		return pflag.NormalizedName(name)
	})
	pflag.CommandLine.AddFlagSet(f)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	checkErr(pflag.Set("logtostderr", "true"))
	checkErr(pflag.CommandLine.MarkHidden("log-flush-frequency"))
	checkErr(pflag.CommandLine.MarkHidden("alsologtostderr"))
	checkErr(pflag.CommandLine.MarkHidden("log-backtrace-at"))
	checkErr(pflag.CommandLine.MarkHidden("log-dir"))
	checkErr(pflag.CommandLine.MarkHidden("logtostderr"))
	checkErr(pflag.CommandLine.MarkHidden("stderrthreshold"))
	checkErr(pflag.CommandLine.MarkHidden("vmodule"))
	// Init logging
	log.Init()
	defer log.Flush()
}

func checkErr(err error) {
	if err != nil {
		glog.Fatalf("error: %v", err)
	}
}

func runCmd(cmd *cobra.Command, _ []string) {
	if f, _ := cmd.Flags().GetBool(FlagVersion); f {
		fmt.Printf("Build version: %s\n", build.Version)
		fmt.Printf("Build timestamp: %s\n", build.Timestamp)
		return
	}

	cf, err := cmd.Flags().GetString(FlagConfigFileName)
	if err != nil {
		glog.Fatalf("can't get config file name: %v", err)
	}

	if cf != "" {
		cf = strings.TrimSpace(cf)

		glog.V(2).Infof("using config file: %s", cf)

		if err := readConfigFile(cmd, cf); err != nil {
			glog.Fatalf("can't read config file: %v", err)
		}

		appCfg, err := newAppConfigFromFile()
		if err != nil {
			glog.Fatalf("config file error: %v", err)
		}

		once, err := cmd.Flags().GetBool(FlagRunOnce)
		if err != nil {
			glog.Fatal(err)
		}

		if once {
			runAppOnce(appCfg)
		} else {
			cfgCheckInterval := viper.GetDuration(ConfigCheckInterval)
			if cfgCheckInterval == 0 {
				glog.Fatal("config file update check interval can't be 0")
			}
			glog.V(2).Infof("will check config file for updates every %v", cfgCheckInterval)
			runApp(appCfg, cfgCheckInterval)
		}
	} else {
		// No config file specified, use flags only for config creating
		appCfg, err := newAppConfigFromFlags(cmd.Flags())
		if err != nil {
			glog.Fatalf("error: %v", err)
		}

		runAppOnce(appCfg)
	}
}

func runAppOnce(appCfg *app.AppConfig) {
	newApp := app.NewApp(appCfg)
	newApp.RunOnce()
}

func runApp(appCfg *app.AppConfig, cfgCheckInterval time.Duration) {
	cfgFile := viper.ConfigFileUsed()

	cfgFileStat, err := os.Stat(cfgFile)
	if err != nil {
		glog.Fatalf("can't stat config file: %v", err)
	}

	cfgCh := make(chan *app.AppConfig)
	doneCh := make(chan struct{})
	signalCh := make(chan os.Signal, 1)

	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	newApp := app.NewApp(appCfg)
	go newApp.Run(cfgCh, doneCh)

Free:
	for {
		select {
		case sig := <-signalCh:
			// Signal received, close config channel and wait for app stopping
			glog.Infof("received %v signal", sig)
			close(cfgCh)
			<-doneCh
			break Free // I want to :)
		case <-time.After(cfgCheckInterval):
			s, err := os.Stat(cfgFile)
			if err != nil {
				glog.Errorf("can't stat config file: %v", err)
				continue
			}
			if cfgFileStat.Size() == s.Size() && !cfgFileStat.ModTime().Before(s.ModTime()) {
				glog.V(4).Infof("config file is unchanged")
				continue
			}
			glog.Infof("config file is updated")
			cfgFileStat = s
			if err := viper.ReadInConfig(); err != nil {
				glog.Errorf("can't read config file: %v", err)
				continue
			}
			newCfgCheckInterval := viper.GetDuration(ConfigCheckInterval)
			if newCfgCheckInterval == 0 {
				glog.Errorf("invalid new config file check interval: %v", newCfgCheckInterval)
			} else if newCfgCheckInterval != cfgCheckInterval {
				cfgCheckInterval = newCfgCheckInterval
				glog.V(2).Infof("config file check interval changed to %v", cfgCheckInterval)
			}
			newAppCfg, err := newAppConfigFromFile()
			if err != nil {
				glog.Errorf("config file error: %v", err)
				continue
			}
			// Notify app about config file update
			cfgCh <- newAppCfg
		}
	}

	glog.V(2).Info("exiting")
}

func newAppConfigFromFlags(f *pflag.FlagSet) (*app.AppConfig, error) {
	chainName, err := f.GetString(FlagIpChainName)
	if err != nil {
		return nil, err
	}

	ports, err := f.GetStringSlice(FlagRestrictedPorts)
	if err != nil {
		return nil, err
	}
	if ports == nil || len(ports) == 0 {
		return nil, errors.New(fmt.Sprintf("no restricted ports defined (use '--%s' option)", FlagRestrictedPorts))
	}
	if err := util.ValidatePorts(ports); err != nil {
		return nil, err
	}

	nets, err := f.GetStringSlice(FlagAllowedNetworks)
	if err != nil {
		return nil, err
	}
	if nets == nil || len(nets) == 0 {
		return nil, errors.New(fmt.Sprintf("no allowed networks defined (use '--%s' option)", FlagAllowedNetworks))
	}
	if err := util.ValidateNetworks(nets); err != nil {
		return nil, err
	}

	glog.V(2).Infof("chain name: %s, restricted ports: %v, allowed networks: %v", chainName, ports, nets)

	return app.NewAppConfig(chainName, ports, nets), nil
}

func newAppConfigFromFile() (*app.AppConfig, error) {
	chainName := viper.GetString(ConfigIpChainName)

	ports := viper.GetStringSlice(ConfigRestrictedPorts)
	if ports == nil || len(ports) == 0 {
		return nil, errors.New(fmt.Sprintf("no restricted ports defined (add '%s' section)", ConfigRestrictedPorts))
	}
	if err := util.ValidatePorts(ports); err != nil {
		return nil, err
	}

	nets := viper.GetStringSlice(ConfigAllowedNetworks)
	if nets == nil || len(nets) == 0 {
		return nil, errors.New(fmt.Sprintf("no allowed networks defined (add '%s' section)", ConfigAllowedNetworks))
	}
	if err := util.ValidateNetworks(nets); err != nil {
		return nil, err
	}

	glog.V(2).Infof("chain name: %s, restricted ports: %v, allowed networks: %v", chainName, ports, nets)

	return app.NewAppConfig(chainName, ports, nets), nil
}

func readConfigFile(cmd *cobra.Command, cf string) error {
	viper.SetConfigFile(cf)

	if err := viper.BindPFlag(ConfigCheckInterval, cmd.Flags().Lookup(FlagConfigCheckInterval)); err != nil {
		return err
	}
	if err := viper.BindPFlag(ConfigIpChainName, cmd.Flags().Lookup(FlagIpChainName)); err != nil {
		return err
	}
	if err := viper.BindPFlag(ConfigRestrictedPorts, cmd.Flags().Lookup(FlagRestrictedPorts)); err != nil {
		return err
	}
	if err := viper.BindPFlag(ConfigAllowedNetworks, cmd.Flags().Lookup(FlagAllowedNetworks)); err != nil {
		return err
	}

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}
