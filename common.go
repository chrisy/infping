// common stuff for infping and infhttp
// Copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

func herr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func perr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func lookupAddrFromIface(iface string, ipv6 bool) (net.IP, error) {
	i, err := net.InterfaceByName(iface)
	herr(err)

	addrs, err := i.Addrs()
	herr(err)

	var ip net.IP
	for _, addr := range addrs {

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		addrstr := ip.String()
		if ipv6 {
			if strings.Contains(addrstr, ":") {
				if strings.HasPrefix(addrstr, "f") {
					// link local or something
					continue
				}
				return ip, nil
			} else {
				// want ipv6, no ipv6 addr!
				continue
			}
		} else {
			return ip, nil
		}
	}

	// no suitable address found
	err = errors.New("no suitable IP address found on interface")
	return ip, err
}
