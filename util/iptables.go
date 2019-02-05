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
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const (
	restrictedPortsInputRuleId = "kube-restrict-ip"

	restrictedPortsInputRuleRegexTemplate = "^-A INPUT -p tcp -m multiport --dports ([0-9,]+) -m comment --comment \"?" +
		restrictedPortsInputRuleId + "\"? -j %s$"
)

func CreateEmptyChainRule(chainName string) string {
	return fmt.Sprintf(":%s - [0:0]", chainName)
}

func CreateRestrictedPortsAddRule(chain string, ports []string) string {
	return JoinWords("-I", "INPUT", "1", CreateRestrictedPortsMatchRule(chain, ports))
}

func CreateRestrictedPortsDeleteRule(chain string, ports []string) string {
	return JoinWords("-D", "INPUT", CreateRestrictedPortsMatchRule(chain, ports))
}

func CreateRestrictedPortsMatchRule(chain string, ports []string) string {
	p := strings.Join(ports, ",")
	return JoinWords("-p", "tcp", "-m", "multiport", "--dports", p,
		"-m", "comment", "--comment", "\""+restrictedPortsInputRuleId+"\"", "-j", chain)
}

func CreateAllowedNetworkChainRule(chain string, net string) string {
	return JoinWords("-A", chain, "-s", net, "-j", "RETURN")
}

func CreateDefaultNetworkChainRule(chain string) string {
	return JoinWords("-A", chain, "-j", "REJECT", "--reject-with", "icmp-port-unreachable")
}

// Validate slice of IP port numbers in string form
func ValidatePorts(ports []string) error {
	for _, p := range ports {
		if n, err := strconv.Atoi(p); err != nil || n <= 0 || n > 65535 {
			return errors.New(fmt.Sprintf("invalid port: %s", p))
		}
	}
	return nil
}

// Validate slice of IP networks (addresses or CIDRs)
func ValidateNetworks(nets []string) error {
	for _, n := range nets {
		if _, _, err := net.ParseCIDR(n); err != nil && net.ParseIP(n) == nil {
			return errors.New(fmt.Sprintf("invalid network: %s", n))
		}
	}
	return nil
}

func GetRestrictedPortsFromTablesData(data []byte, chain string) []string {
	if data == nil || len(data) == 0 {
		return nil
	}

	lines := string(data)
	re := regexp.MustCompile(fmt.Sprintf(restrictedPortsInputRuleRegexTemplate, chain))
	for _, line := range strings.Split(lines, "\n") {
		m := re.FindStringSubmatch(line)
		if m != nil {
			return strings.Split(m[1], ",")
		}
	}

	return nil
}
