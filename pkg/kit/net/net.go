package net

import (
	"endpoint/pkg/zlog"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"
)

var (
	address string
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		p := r.Intn(50001) + 10000
		if !CheckPortAvailability(p) {
			return p
		}
	}
}

func CheckPortAvailability(port int) (flag bool) {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return !flag
	}
	defer func() {
		if listen != nil {
			zlog.Unwrap(listen.Close())
		}
	}()

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	listenUDP, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return !flag
	}
	defer func() {
		if listenUDP != nil {
			zlog.Unwrap(listenUDP.Close())
		}
	}()

	return flag
}

func GetTcpListener() (net.Listener, int, error) {
	port := GetFreePort()
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, 0, err
	}

	zlog.Info(fmt.Sprintf("listening TCP on [%s]%s", l.Addr().Network(), l.Addr().String()))

	return l, port, nil
}

func GetUdpListener() (*net.UDPConn, int, error) {
	port := GetFreePort()
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, 0, err
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, 0, err
	}

	zlog.Info(fmt.Sprintf("listening UDP in [%s]%s", l.LocalAddr().Network(), l.LocalAddr().String()))

	return l, port, nil
}
