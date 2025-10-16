# coredns-ipset

The plugin for adding resolved IP addresses to ipset (must be installed) or nft set lists. Helps route traffic for domains (and domain suffixes, e.g., .ru will add all ips from .ru domains to desired ipset) through the desired gateway. It supports ipv4 and ipv6 (disabled by default) ipset and nft set lists.

## Building

```bash
git clone https://github.com/coredns/coredns
cd coredns
sed -i.bak -r '/ipset:.*/d' plugin.cfg
sed -i.bak '/forward:.*/i ipset:github.com/foi/coredns-ipset' plugin.cfg
go get github.com/foi/coredns-ipset
go mod tidy
go generate
make
```

Alternatively, you can download pre-built binaries from the [releases](https://github.com/foi/coredns-ipset/releases/) page for amd64 or arm64.

For development, clone the coredns-ipset repository and add to the go.mod file of the [coredns](https://github.com/coredns/coredns) repository: `replace github.com/foi/coredns-ipset => ../coredns-ipset`, then run `go mod tidy`, `go get github.com/foi/coredns-ipset`, `go mod tidy`, `go generate`, and `make`.

## Prepare binary

If coredns is planned to listen on port 53, it is necessary to add capabilities to the binary:

```bash
sudo setcap 'cap_net_bind_service,cap_net_admin=+ep' /usr/local/bin/coredns
```

## Example configuration

### ipset

```
. {
  # some resolvers config
  ....
  ipset {
      # This setting enables IPv6 ipset lists.
      # By default, only IPv4 ipset lists are used.
      # The ipset lists must be created in advance, and IPv6 lists should have an -ipv6 suffix.
      ipv6
      russia-ipset-name listofdomains.txt anotherlist.txt
      usa-ipset-name listofdomains2.txt
  }
}
```
### nftables

```
. {
  # some resolvers config
  ...

  ipset {
      # This setting enables IPv6 nft set lists.
      # By default, only IPv4 nft set lists are used.
      # The nft set lists must be created in advance, and IPv6 lists should have an -ipv6 suffix.
      # directives' order is important!
      ipv6
      nft mytable:inet mytable2:inet
      russia-nft-set-name listofdomains.txt listofdomains3.txt
      usa-nft-set-name listofdomains2.txt
  }
}
```
If you don't specify ":inet", it will try to use ipv4 and ipv6 table types (if enabled).

You can find in [examples](https://github.com/foi/coredns-ipset/tree/main/examples) repo's folder.

## ROADMAP

- [ ] compact build for openwrt
- [ ] installation instruction for openwrt
