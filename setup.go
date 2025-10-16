package ipset

import (
	"os"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/google/nftables"
	"github.com/vishvananda/netlink"
)

//nolint:gochecknoinits // coredns plugins requires init()
func init() { plugin.Register("ipset", setup) }

func setup(c *caddy.Controller) error {
	ipsetLists := map[string][]string{}
	ipsetType := 0
	nftablesTableNames := []string{}
	var ipv6Enabled bool

	for c.Next() {
		for c.NextBlock() {
			firstToken := c.Val()
			switch firstToken {
			case "ipv6":
				ipv6Enabled = true
			case "nft":
				ipsetType = IpsetNft
				nftablesTableNames = c.RemainingArgs()
			default:
				var err error

				if ipsetType == IpsetClassic {
					if err != nil {
						log.Fatalf(
							"ipset list %s, err: %s",
							firstToken,
							err,
						)
					}

					if ipv6Enabled {
						_, err = netlink.IpsetList(firstToken + "-ipv6")
						if err != nil {
							log.Fatalf(
								"ipset list %s, err: %s",
								firstToken+"-ipv6",
								err,
							)
						}
					}
				}

				domainsListsPaths := c.RemainingArgs()
				log.Infof(
					"processing ipset list %s with domain lists %s",
					firstToken,
					domainsListsPaths,
				)

				for _, listPath := range domainsListsPaths {
					_, err = os.Stat(listPath)
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
					_, ok := ipsetLists[firstToken]
					if !ok {
						ipsetLists[firstToken] = []string{}
					}

					for _, domain := range domains {
						ipsetLists[firstToken] = append(
							ipsetLists[firstToken],
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

		if ipsetType == IpsetClassic {
			log.Info("ipset mode enabled")
		} else {
			log.Info("nftables mode enabled")
		}
	}

	dnsserver.GetConfig(c).
		AddPlugin(func(next plugin.Handler) plugin.Handler {
			ipsetPluginStruct := Ipset{
				Next:                     next,
				ipsetListDomainNamesList: ipsetLists,
				ResolvedIps:              map[string]struct{}{},
				IPv6Enabled:              ipv6Enabled,
				IpsetType:                ipsetType,
			}

			//nolint: nestif // todo
			if ipsetType == IpsetNft {
				nftcon, err := nftables.New()
				if err != nil {
					log.Fatalf("failed to init to nft connection, err: %s", err.Error())
				}

				ipsetPluginStruct.NftCon = nftcon
				nftablesTables := []*nftables.Table{}
				nftTablesActual, err := nftcon.ListTables()
				if err != nil {
					log.Fatal(err.Error())
				}
				for _, tableName := range nftablesTableNames {
					if strings.HasSuffix(tableName, ":inet") {
						table := &nftables.Table{
							Name:   strings.Split(tableName, ":inet")[0],
							Family: nftables.TableFamilyINet,
						}

						if findNftTable(table, nftTablesActual, nftables.TableFamilyINet) {
							nftablesTables = append(
								nftablesTables,
								table,
							)
						} else {
							log.Fatalf("missing inet nft table %s", tableName)
						}

						continue
					}

					tableIPv4 := &nftables.Table{
						Name:   tableName,
						Family: nftables.TableFamilyIPv4,
					}

					if findNftTable(tableIPv4, nftTablesActual, nftables.TableFamilyIPv4) {
						nftablesTables = append(
							nftablesTables,
							tableIPv4,
						)
					} else {
						log.Fatalf("missing ipv4 nft table %s", tableName)
					}

					if ipv6Enabled {
						tableIPv6 := &nftables.Table{
							Name:   tableName + "-ipv6",
							Family: nftables.TableFamilyIPv6,
						}

						if findNftTable(tableIPv6, nftTablesActual, nftables.TableFamilyIPv6) {
							nftablesTables = append(
								nftablesTables,
								tableIPv4,
							)
						} else {
							log.Fatalf("missing ipv6 nft table %s", tableName)
						}
					}
				}

				ipsetPluginStruct.NftablesTables = nftablesTables

				for _, nftable := range ipsetPluginStruct.NftablesTables {
					log.Debugf("using nft table %s family type %d", nftable.Name, nftable.Family)
				}
			} else {
				ipsetPluginStruct.NftCon = nil
			}

			return ipsetPluginStruct
		})

	return nil
}

func findNftTable(table *nftables.Table, nftTablesActual []*nftables.Table, family nftables.TableFamily) bool {
	for _, nfttable := range nftTablesActual {
		if table.Name == nfttable.Name && table.Family == family {
			return true
		}
	}

	return false
}
