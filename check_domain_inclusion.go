package ipset

import "strings"

func (e Ipset) checkDomainInclusion(
	domainName string,
) []string {
	matchedIpsetsListsNames := []string{}
	for k := range e.ipsetListDomainNamesList {
		log.Debugf(
			"trying to find %s in %s ipset list",
			domainName,
			k,
		)

		for _, domain := range e.ipsetListDomainNamesList[k] {
			if len(domain) != 0 {
				if strings.HasSuffix(
					domainName,
					domain,
				) || strings.TrimPrefix(domain, ".") == domainName {
					log.Debugf(
						"%s has found in %s ipset list",
						domainName,
						k,
					)
					matchedIpsetsListsNames = append(
						matchedIpsetsListsNames,
						k,
					)
				}
			}
		}
	}
	return matchedIpsetsListsNames
}
