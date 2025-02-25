package V2RayAPI

import (
	"context"
	"github.com/crossfw/Air-Universe/pkg/structures"
	"github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/app/proxyman"
	"github.com/v2fly/v2ray-core/v4/app/proxyman/command"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/protocol/tls/cert"
	"github.com/v2fly/v2ray-core/v4/common/serial"
	"github.com/v2fly/v2ray-core/v4/infra/conf"
	ssInbound "github.com/v2fly/v2ray-core/v4/proxy/shadowsocks"
	trojanInbound "github.com/v2fly/v2ray-core/v4/proxy/trojan"
	vmessInbound "github.com/v2fly/v2ray-core/v4/proxy/vmess/inbound"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tcp"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v4/transport/internet/websocket"
)

func addInbound(client command.HandlerServiceClient, node *structures.NodeInfo) error {
	var (
		protocolName      string
		transportSettings []*internet.TransportConfig
		securityType      string
		securitySettings  []*serial.TypedMessage
		proxySetting      *serial.TypedMessage
	)
	switch node.TransportMode {
	case "ws":
		protocolName = "websocket"
		if node.Path == "" {
			node.Path = "/"
		}
		header := []*websocket.Header{
			{
				Key:   "Host",
				Value: node.Host,
			},
		}

		transportSettings = []*internet.TransportConfig{
			{
				ProtocolName: protocolName,
				Settings: serial.ToTypedMessage(&websocket.Config{
					Path:                node.Path,
					Header:              header,
					AcceptProxyProtocol: node.EnableProxyProtocol,
				},
				),
			},
		}

	case "tcp":
		protocolName = "tcp"
		transportSettings = []*internet.TransportConfig{
			{
				ProtocolName: protocolName,
				Settings: serial.ToTypedMessage(&tcp.Config{
					AcceptProxyProtocol: node.EnableProxyProtocol,
				}),
			},
		}
	}

	if node.EnableTLS == true && node.Cert.CertPath != "" && node.Cert.KeyPath != "" {
		// Use custom cert file
		certConfig := &conf.TLSCertConfig{
			CertFile: node.Cert.CertPath,
			KeyFile:  node.Cert.KeyPath,
		}
		builtCert, _ := certConfig.Build()
		securityType = serial.GetMessageType(&tls.Config{})
		securitySettings = []*serial.TypedMessage{
			serial.ToTypedMessage(&tls.Config{
				Certificate: []*tls.Certificate{builtCert},
			}),
		}
	} else if node.EnableTLS == true {
		// Auto build cert
		securityType = serial.GetMessageType(&tls.Config{})
		securitySettings = []*serial.TypedMessage{
			serial.ToTypedMessage(&tls.Config{
				Certificate: []*tls.Certificate{tls.ParseCertificate(cert.MustGenerate(nil))},
			}),
		}
	} else {
		// Disable TLS
		securityType = ""
		securitySettings = nil
	}

	switch node.Protocol {
	case "vmess":
		proxySetting = serial.ToTypedMessage(&vmessInbound.Config{
			Detour: &vmessInbound.DetourConfig{
				To: "direct",
			},
		})
	case "trojan":
		proxySetting = serial.ToTypedMessage(&trojanInbound.ServerConfig{})
	case "ss":
		proxySetting = serial.ToTypedMessage(&ssInbound.ServerConfig{
			Network: []net.Network{2, 3},
		})
	}

	_, err := client.AddInbound(context.Background(), &command.AddInboundRequest{
		Inbound: &core.InboundHandlerConfig{
			Tag: node.Tag,
			ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
				PortRange: net.SinglePortRange(net.Port(node.ListenPort)),
				Listen:    net.NewIPOrDomain(net.AnyIP),
				SniffingSettings: &proxyman.SniffingConfig{
					Enabled:             true,
					DestinationOverride: []string{"http", "tls"},
				},
				StreamSettings: &internet.StreamConfig{
					ProtocolName:      protocolName,
					TransportSettings: transportSettings,
					SecurityType:      securityType,
					SecuritySettings:  securitySettings,
				},
			}),
			ProxySettings: proxySetting,
		},
	})

	return err
}

func removeInbound(client command.HandlerServiceClient, node *structures.NodeInfo) error {
	_, err := client.RemoveInbound(context.Background(), &command.RemoveInboundRequest{
		Tag: node.Tag,
	})
	return err
}
