# coredns-ipset

The plugin for adding resolved IP addresses to ipset (must be installed) lists. Helps route traffic for domains through the desired gateway. It supports ipv4 and ipv6 (disabled by default) ipset lists.

## Building

```bash
git clone https://github.com/coredns/coredns
cd coredns
sed -i.bak -r '/ipset:.*/d' plugin.cfg
sed -i.bak '/forward:.*/i ipset:github.com/foi/coredns-ipset' plugin.cfg
go get github.com/foi/coredns-ipset
go generate
make
```

Alternatively, you can download pre-built binaries from the [releases](https://github.com/foi/coredns-ipset/releases/) page for amd64 or arm64.

For development, clone the coredns-ipset repository and add to the go.mod file of the [coredns](https://github.com/coredns/coredns) repository: `replace github.com/foi/coredns-ipset => ../coredns-ipset`, then run `go mod tidy`, `go get github.com/foi/coredns-ipset`, `go generate`, and `make`.

## Prepare binary

If coredns is planned to listen on port 53, it is necessary to add capabilities to the binary:

```bash
sudo setcap 'cap_net_bind_service,cap_net_admin=+ep' /usr/local/bin/coredns
```

## Example configuration

```
. {
  ipset {
      # This setting enables IPv6 ipset lists.
      # By default, only IPv4 ipset lists are used.
      # The ipset lists must be created in advance, and IPv6 lists should have an -ipv6 suffix.
      ipv6
      russia-ipset-name listofdomains.txt
      usa-ipset-name listofdomains2.txt
  }
}
```

You can find in [examples](https://github.com/foi/coredns-ipset/tree/main/examples) repo's folder.
