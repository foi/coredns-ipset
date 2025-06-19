package ipset

import (
	"os"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/vishvananda/netlink"
)

func init() { plugin.Register("ipset", setup) }

func setup(c *caddy.Controller) error {
	ipsetLists := map[string][]string{}
	var ipv6Enabled bool
	for c.Next() {
		for c.NextBlock() {
			ipsetListName := c.Val()
			if ipsetListName == "ipv6" {
				ipv6Enabled = true
			} else {
				_, err := netlink.IpsetList(ipsetListName)
				if err != nil {
					log.Fatalf(
						"ipset list %s, err: %s",
						ipsetListName,
						err,
					)
				}
				if ipv6Enabled {
					_, err := netlink.IpsetList(ipsetListName + "-ipv6")
					if err != nil {
						log.Fatalf(
							"ipset list %s, err: %s",
							ipsetListName+"-ipv6",
							err,
						)
					}
				}

				domainsListsPaths := c.RemainingArgs()
				log.Infof(
					"processing ipset list %s with domain lists %s",
					ipsetListName,
					domainsListsPaths,
				)

				for _, listPath := range domainsListsPaths {
					_, err := os.Stat(listPath)
					if err != nil {
						log.Fatalf(
							"stat domains list file, err: %s",
							err,
						)
					}
					domainListFileContent, err := os.ReadFile(
						listPath,
					)
					if err != nil {
						log.Fatalf(
							"read file domains list file, err: %s",
							err,
						)
					}
					log.Debugf(
						"%+v",
						string(domainListFileContent),
					)
					domains := strings.Split(
						string(domainListFileContent),
						"\n",
					)
					_, ok := ipsetLists[ipsetListName]
					if !ok {
						ipsetLists[ipsetListName] = []string{}
					}
					for _, domain := range domains {
						ipsetLists[ipsetListName] = append(
							ipsetLists[ipsetListName],
							"."+domain,
						)
					}
				}
			}
		}

		log.Debugf(
			"ipset lists contents: %+v",
			ipsetLists,
		)
	}

	dnsserver.GetConfig(c).
		AddPlugin(func(next plugin.Handler) plugin.Handler {
			return Ipset{
				Next:                     next,
				ipsetListDomainNamesList: ipsetLists,
				ResolvedIps:              map[string]struct{}{},
				IPv6Enabled:              ipv6Enabled,
			}
		})

	return nil
}
