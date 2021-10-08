package plugin

import (
	"fmt"
	"strconv"

	"github.com/fatedier/frp/test/e2e/framework"
	"github.com/fatedier/frp/test/e2e/framework/consts"
	"github.com/fatedier/frp/test/e2e/pkg/port"
	"github.com/fatedier/frp/test/e2e/pkg/request"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("[Feature: Client-Plugins]", func() {
	f := framework.NewDefaultFramework()

	Describe("UnixDomainSocket", func() {
		It("Expose a unix domain socket echo server", func() {
			serverConf := consts.DefaultServerConfig
			clientConf := consts.DefaultClientConfig

			getProxyConf := func(proxyName string, portName string, extra string) string {
				return fmt.Sprintf(`
				[%s]
				type = tcp
				remote_port = {{ .%s }}
				plugin = unix_domain_socket
				plugin_unix_path = {{ .%s }}
				`+extra, proxyName, portName, framework.UDSEchoServerAddr)
			}

			tests := []struct {
				proxyName   string
				portName    string
				extraConfig string
			}{
				{
					proxyName: "normal",
					portName:  port.GenName("Normal"),
				},
				{
					proxyName:   "with-encryption",
					portName:    port.GenName("WithEncryption"),
					extraConfig: "use_encryption = true",
				},
				{
					proxyName:   "with-compression",
					portName:    port.GenName("WithCompression"),
					extraConfig: "use_compression = true",
				},
				{
					proxyName: "with-encryption-and-compression",
					portName:  port.GenName("WithEncryptionAndCompression"),
					extraConfig: `
					use_encryption = true
					use_compression = true
					`,
				},
			}

			// build all client config
			for _, test := range tests {
				clientConf += getProxyConf(test.proxyName, test.portName, test.extraConfig) + "\n"
			}
			// run frps and frpc
			f.RunProcesses([]string{serverConf}, []string{clientConf})

			for _, test := range tests {
				framework.NewRequestExpect(f).Port(f.PortByName(test.portName)).Ensure()
			}
		})
	})

	It("plugin http_proxy", func() {
		serverConf := consts.DefaultServerConfig
		clientConf := consts.DefaultClientConfig

		remotePort := f.AllocPort()
		clientConf += fmt.Sprintf(`
		[tcp]
		type = tcp
		remote_port = %d
		plugin = http_proxy
		plugin_http_user = abc
		plugin_http_passwd = 123
		`, remotePort)

		f.RunProcesses([]string{serverConf}, []string{clientConf})

		// http proxy, no auth info
		framework.NewRequestExpect(f).PortName(framework.HTTPSimpleServerPort).RequestModify(func(r *request.Request) {
			r.HTTP().Proxy("http://127.0.0.1:" + strconv.Itoa(remotePort))
		}).Ensure(framework.ExpectResponseCode(407))

		// http proxy, correct auth
		framework.NewRequestExpect(f).PortName(framework.HTTPSimpleServerPort).RequestModify(func(r *request.Request) {
			r.HTTP().Proxy("http://abc:123@127.0.0.1:" + strconv.Itoa(remotePort))
		}).Ensure()

		// connect TCP server by CONNECT method
		framework.NewRequestExpect(f).PortName(framework.TCPEchoServerPort).RequestModify(func(r *request.Request) {
			r.TCP().Proxy("http://abc:123@127.0.0.1:" + strconv.Itoa(remotePort))
		})
	})
})
