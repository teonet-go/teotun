# Teonet tunnel

Teotun creates secret tunnel between hosts without public IPs using [Teonet](https://github.com/teonet-go/teonet). The connection based on [TRU](https://github.com/teonet-go/tru) transport and create reliable, low latency, encrypted P2P channels between connected peers.

[![GoDoc](https://godoc.org/github.com/teonet-go/teotun?status.svg)](https://godoc.org/github.com/teonet-go/teotun/)
[![Go Report Card](https://goreportcard.com/badge/github.com/teonet-go/teotun)](https://goreportcard.com/report/github.com/teonet-go/teotun)

## Usage example

Create regular tunnel between thee hosts.

One host will be Main and all other will connect to main host on start. Main host does not have -connectto parameter. All other hosts use teonet address of Main host in -connectto parameter.

### For any hosts

Connect to your host and clone this reposipory:

```shell
git clone https://github.com/teonet-go/teotun.git
cd teotun
```

### Start teotun on Main host

```shell
TUN=teotun1 && sudo go run ./cmd/teotun/ -name=$TUN -postcon="./if_up.sh $TUN 10.1.2.1/24" -loglevel=connect -hotkey -stat
```

Copy teonet address which prints after Main teotun started:

```
Teonet address: MIxxCM5mxilJ9Oa4zvQJbkSBp7mB4xuyZMM
```

Use this address in -connectto parameter in [Host A](start-teotun-in-host-a) and [Host B](start-teotun-in-host-b)

### Start teotun in Host A

```shell
TUN=teotun1 && sudo go run ./cmd/teotun/ -name=$TUN -connectto=MIxxCM5mxilJ9Oa4zvQJbkSBp7mB4xuyZMM -postcon="./if_up.sh $TUN 10.1.2.2/24" -loglevel=connect -hotkey -stat
```

### Start teotun in Host B

```shell
TUN=teotun1 && sudo go run ./cmd/teotun/ -name=$TUN -connectto=MIxxCM5mxilJ9Oa4zvQJbkSBp7mB4xuyZMM -postcon="./if_up.sh $TUN 10.1.2.3/24" -loglevel=connect -hotkey -stat
```

### How to use

When teotun will be started on all hosts, you can use any network commands between this hosts by its local IPs 10.1.2.1, 10.1.2.2, 10.1.2.3.

For example, you can ping [Host B](start-teotun-in-host-b) from [Host A](start-teotun-in-host-a).

Login to [Host B](start-teotun-in-host-b) and execute command:

```shell
ping 10.1.2.2
```

All host in teotun network connect P2P so you will see lowest ping between [Host B](start-teotun-in-host-b) and [Host A](start-teotun-in-host-a).

## How it works

_This article is Under construction yet..._

## License

[BSD](LICENSE)
