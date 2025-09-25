package ipset

import (
	"net"

	"github.com/google/nftables"
	"github.com/vishvananda/netlink"
)

func (e Ipset) AddIPToSet(ip net.IP, ipsetName string) error {
	if e.IpsetType == IpsetNft {
		log.Debugf(
			"adding %s to nft set %s",
			ip.String(),
			ipsetName,
		)

		ips := []nftables.SetElement{
			{
				Key: ip,
			},
		}

		for _, table := range e.NftablesTables {
			err := e.NftCon.SetAddElements(
				&nftables.Set{
					Table: table,
					Name:  ipsetName,
				},
				ips,
			)
			if err != nil {
				return err
			}
		}

		return e.NftCon.Flush()
	}

	log.Debugf(
		"adding %s to ipset %s",
		ip.String(),
		ipsetName,
	)

	return netlink.IpsetAdd(
		ipsetName,
		&netlink.IPSetEntry{
			IP: ip,
		},
	)
}
