package main

import (
	"fmt"
	"net"
)

func getMyIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func usage() {
	fmt.Println("****************************************")
	fmt.Println(usagePrivate)
	fmt.Println(usageChangeNick)
	fmt.Println(usageExit)
	fmt.Println(usageGetNicks)
	fmt.Println("****************************************")
}
