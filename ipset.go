package ipset

import (
	"context"
	"fmt"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/miekg/dns"
	"github.com/vishvananda/netlink"
)

var log = clog.NewWithPlugin("ipset")

type Ipset struct {
	Next                     plugin.Handler
	ipsetListDomainNamesList map[string][]string
	ResolvedIps              map[string]struct{}
	IPv6Enabled              bool
}

func (e Ipset) ServeDNS(
	ctx context.Context,
	w dns.ResponseWriter,
	r *dns.Msg,
) (int, error) {
	nw := nonwriter.New(w)
	rcode, err := plugin.NextOrFailure(
		e.Name(),
		e.Next,
		ctx,
		nw,
		r,
	)
	if err != nil {
		return rcode, err
	}
	r = nw.Msg
	if r == nil {
		return dns.RcodeFormatError, fmt.Errorf(
			"no answer received",
		)
	}

	log.Debugf(
		"%+v, %+v, %+v",
		r.Answer,
		r.Question,
		rcode,
	)

	requestedDomain := strings.TrimSuffix(
		r.Question[0].Name,
		".",
	)
	matchedIpsetLists := e.checkDomainInclusion(
		requestedDomain,
	)

	log.Debugf(
		"%s domain found in %d ipsets lists",
		requestedDomain,
		len(matchedIpsetLists),
	)

	for _, answer := range r.Answer {
		switch rr := answer.(type) {
		case *dns.A:
			if len(matchedIpsetLists) != 0 {
				for _, ipsetListName := range matchedIpsetLists {
					_, exist := e.ResolvedIps[rr.A.String()]
					if rr.A != nil && !rr.A.IsUnspecified() && !exist {
						err = netlink.IpsetAdd(ipsetListName, &netlink.IPSetEntry{IP: rr.A.To4()})
						if err != nil {
							log.Errorf(
								"Error while appending %s in ipset '%s' list, err: %s", rr.A.To4().String(),
								ipsetListName,
								err.Error(),
							)
						}

						e.ResolvedIps[rr.A.String()] = struct{}{}
					}
				}
			}
		case *dns.AAAA:
			if e.IPv6Enabled {
				if len(matchedIpsetLists) != 0 {
					for _, ipsetListName := range matchedIpsetLists {
						_, exist := e.ResolvedIps[rr.AAAA.String()]
						if rr.AAAA != nil && !rr.AAAA.IsUnspecified() && !exist {
							err = netlink.IpsetAdd(ipsetListName+"-ipv6", &netlink.IPSetEntry{IP: rr.AAAA.To16()})
							if err != nil {
								log.Errorf(
									"Error while appending %s in ipset '%s' list, err: %s", rr.AAAA.To16().String(),
									ipsetListName+"-ipv6",
									err.Error(),
								)
							}

							e.ResolvedIps[rr.AAAA.String()] = struct{}{}
						}
					}
				}
			}
		}
	}

	_ = w.WriteMsg(r)

	return rcode, nil
}

func (e Ipset) Name() string { return "ipset" }
