# coredns-ipset

Плагин coredns для добавления отрезолвленных ip-адресов в ipset (должен быть установлен) списки. Помогает направлять траффик для доменов через желаемый шлюз.

**IPv6, пока что, не поддерживается**

Пример конфигурации:

В ipset список будут добавлены ip самого домена и все его поддомены

```
# youtube.txt
# ютюб работает из российских датацентров - можно направять траффик из РФ

ytimg.com
ggpht.com
googlevideo.com
play.google.com
youtube.com
doubleclick.net
youtu.be
l.google.com
youtubei.googleapis.com
nhacmp3youtube.com
googleusercontent.com

# blocked-for-russian-ips.txt
# Сервисы что недоступны для ip РФ или заблочены РКН

chatgpt.com
grok.com
cdn.oaistatic.com
grok.com
hetzner.com

```
Подключение в `Corefile`:

Первый аргумент - имя ipset списка, второй и n-аргументы списки доменов

```
. {
    # не отдаем ipv6, если его нет
    template IN AAAA {
        rcode NXDOMAIN
    }

    log
    errors

    forward . tls://1.1.1.1 {
        tls_servername cloudflare-dns.com
        health_check 5s
    }
    forward . tls://8.8.8.8 {
        tls_servername dns.google
        health_check 5s
    }
    forward . tls://77.88.8.8 {
        tls_servername common.dot.dns.yandex.net
        health_check 5s
    }

    ipset {
        youtube youtube.txt
        non-ru blocked-for-russian-ips.txt /etc/domains/medium.txt и т.д.
    }
    cache 3600
}
```
Списки ipset должны быть созданы заблаговременно (до старта coredns).

# Компиляция coredns c плагином coredns-ipset

Плагин ipset должен быть до плагина forward.

```bash
sed -i.bak -r '/ipset:.*/d' plugin.cfg
sed -i.bak '/forward:.*/i ipset:github.com/foi/coredns-ipset' plugin.cfg
go get github.com/foi/coredns-ipset
go generate
make
```

Или можно взять готовые бинарики на [странице релизов](https://github.com/foi/coredns-ipset/releases) для amd64 или arm64.

Для разработки следует склонировать репозиторий [coredns-ipset](https://github.com/foi/coredns-ipset) и добавить в `go.mod` coredns репы `replace github.com/foi/coredns-ipset => ../coredns-ipset`, `go mod tidy`, `go get github.com/foi/coredns-ipset`, `go generate`, `make`.

# Настройка

Если планируется что coredns будет слушать на 53 порту, необходимо докинуть capabilities бинарику
```bash
sudo setcap 'cap_net_bind_service,cap_net_admin=+ep' /usr/local/bin/coredns

```
### Пример настройки

`/etc/systemd/system/coredns.service`
```ini
[Unit]
Description=CoreDNS DNS server
Documentation=https://coredns.io
After=network.target
After=NetworkManager-wait-online.service
Requires=init-gateways.service
After=init-gateways.service

[Service]
PermissionsStartOnly=true
LimitNOFILE=1048576
LimitNPROC=512
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_NET_ADMIN
AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_NET_ADMIN
NoNewPrivileges=true
User=coredns
WorkingDirectory=/etc/coredns
ExecStart=/usr/local/bin/coredns -conf=/etc/coredns/Corefile
ExecReload=/bin/kill -SIGUSR1 $MAINPID
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
Сервис что инициализирует поключение к шлюзам `init-gateways.service` до старта `coredns.service`
```ini
[Unit]
Description=init gateways
NetworkManager-wait-online.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/0-coredns-prepare.sh
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
```
Пример скрипта подключения к шлюзам `/usr/local/bin/0-coredns-prepare.sh`:
```bash
#!/usr/bin/env bash
ipset -exist create non-ru hash:ip
ipset -exist create youtube hash:ip

ip rule add fwmark 101 table russia
ip rule add fwmark 100 table abroad

ip tunnel add gre-msi-fin mode gre local 192.168.24.4 remote 192.168.24.14 ttl 255
ip addr add 192.168.25.2/30 dev gre-msi-fin
ip link set gre-msi-fin up

ip tunnel add gre-tw-msk mode gre local 192.168.24.4 remote 192.168.24.13 ttl 255
ip addr add 192.168.25.6/30 dev gre-tw-msk
ip link set gre-tw-msk up

# Шлюз для доменов что заблочены в РФ или заблочены самим сервисом по geoip
ip route add default via 192.168.25.1 dev gre-msi-fin table abroad
# шлюз для Ресурсов что работают из РФ
ip route add default via 192.168.25.5 dev gre-tw-msk table russia

iptables -t nat -A POSTROUTING -o gre-msi-fin -m set --match-set non-ru dst -j SNAT --to-source 192.168.25.2
iptables -t mangle -A OUTPUT -m set --match-set non-ru dst -j MARK --set-mark 100

iptables -t nat -A POSTROUTING -o gre-tw-msk -m set --match-set youtube dst -j SNAT --to-source 192.168.25.6
iptables -t mangle -A OUTPUT -m set --match-set youtube dst -j MARK --set-mark 101

```
Таблицы маршрутизации `/usr/share/iproute2/rt_tables`:
```bash
#
# reserved values
#
255     local
254     main
253     default
0       unspec
#
# local
#
#1      inr.ruhep
100 abroad
101 russia
```
На шлюзах добавляем правила маскарада для адресов интернета
```bash
iptables -t nat -A POSTROUTING -d 10.0.0.0/8 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 172.16.0.0/12 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 192.168.0.0/16 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 127.0.0.0/8 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 169.254.0.0/16 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 224.0.0.0/4 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -d 240.0.0.0/4 -o eth0 -j RETURN
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

```
