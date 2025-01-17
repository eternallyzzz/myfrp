package net

import (
	"endpoint/pkg/zlog"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	address  string
	one      sync.Once
	highPort = 60000
	lowPort  = 10000
)

func GetExternalIP() (string, error) {
	if address == "" {
		resp, err := http.DefaultClient.Get("https://ipinfo.io/ip")
		if err != nil {
			return "", err
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		address = string(bytes)
	}
	return address, nil
}

func GetFreePort() int {
	one.Do(func() {
		value := viper.GetString("endpoint.randPort")
		if value != "" {
			split := strings.Split(value, "-")
			low, err := strconv.Atoi(split[0])
			zlog.UnwrapFatal(err)
			high, err := strconv.Atoi(split[1])
			zlog.UnwrapFatal(err)

			if low < high {
				highPort = high
				lowPort = low
			}
		}
	})

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		p := r.Intn(highPort-lowPort) + lowPort
		if !CheckPortAvailability(p) {
			return p
		}
	}
}

func CheckPortAvailability(port int) bool {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	defer listen.Close()

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	listenUDP, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return true
	}
	defer listenUDP.Close()

	return false
}

func GetTcpListener(port uint16) (net.Listener, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	zlog.Info(fmt.Sprintf("listening TCP on [%s]%s", l.Addr().Network(), l.Addr().String()))

	return l, nil
}

func GetUdpListener(port uint16) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	zlog.Info(fmt.Sprintf("listening UDP on [%s]%s", l.LocalAddr().Network(), l.LocalAddr().String()))

	return l, nil
}
