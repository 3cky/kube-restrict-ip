# kube-restrict-ip

The kube-restrict-ip configures `iptables` rules to restrict access to specified ports of the Kubernetes nodes to defined set of IP addresses.

It creates an `iptables` app chain called `KUBE-RESTRICT-IP` (could be configured), which contains match rules for user-specified IP addresses (hosts and CIDR ranges). It also creates a rule in `INPUT` that jumps to app chain for any traffic bound to restricted ports. All IPs that not match the rules in the app chain are rejected.

## Launching as a DaemonSet

This repo includes an example yaml file that can be used to launch the kube-restrict-ip as a DaemonSet in a Kubernetes cluster.

```
kubectl create -f kube-restrict-ip.yaml
```

The spec in `kube-restrict-ip.yaml` specifies the `kube-system` namespace for the DaemonSet Pods.

## Command Line Options

```
      --allowed-networks strings   allowed networks
  -t, --check-interval duration    config file update check interval (default 60s)
  -c, --config-file string         config file name to watch (implied 'once' if omitted)
  -h, --help                       help for kube-restrict-ip
      --ip-chain string            iptables chain name (default "KUBE-RESTRICT-IP")
      --once                       run once and exit
      --restricted-ports strings   restricted ports
  -v, --v Level                    log level for V logs
  -V, --version                    display the build number and timestamp
```

## Configuration File

kube-restrict-ip looks for YAML or JSON configuration file specified by `--config-file` command line option.

Config file keys:

- `restrictedPorts []int`: A list restricted TCP ports (required).
- `allowedNetworks []string`: A list allowed networks in CIDR notation (required).
- `ipChain string`: iptables chain name (optional, default "KUBE-RESTRICT-IP").
- `checkInterval string`: The interval to check config for updates (optional, default 60s). The syntax is any format accepted by Go's [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration) function.

The docker image of kube-restrict-ip will look for a config file in its container at `/etc/kube-restrict-ip/config.yaml`. This file can be provided via a `ConfigMap`, so it can be reconfigured in a live cluster by creating or editing this `ConfigMap`.

This repo includes an example config file that could be used to create the `ConfigMap` in your cluster:

```
kubectl create configmap kube-restrict-ip --from-file=config.yaml --namespace=kube-system
```

Please note that the `ConfigMap` in the same namespace as the DaemonSet Pods, and named the `kube-restrict-ip` to match the DaemonSet spec. This is necessary for the `ConfigMap` to appear in the Pods' filesystems.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request

## License

kube-restrict-ip is released under the Apache 2.0 license. See [LICENSE.txt](https://github.com/3cky/kube-restrict-ip/blob/master/LICENSE.txt)