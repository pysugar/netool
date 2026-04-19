package distro

import "github.com/pysugar/netool/cmd/base"

func init() {
	base.AddSubCommands(fileServerCmd)
	base.AddSubCommands(httpProxyCmd)
	base.AddSubCommands(registryCmd)
	base.AddSubCommands(discoveryCmd)
	base.AddSubCommands(devtoolCmd)
	base.AddSubCommands(echoServiceCmd)
}
