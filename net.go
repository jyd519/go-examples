package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

func main() {
	infs, err := net.Interfaces()
	if err != nil {
		fmt.Println("get Interfaces: ", err)
		return
	}

	for _, v := range infs {
		if v.Flags&net.FlagUp != net.FlagUp {
			fmt.Println("down", v)
			continue
		}
		if v.Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		if v.Flags&net.FlagBroadcast != 0 {
			fmt.Println("FlagBroadcast", v)
			addrs, err := v.Addrs()
			if err == nil {
				for _, a := range addrs {
					cidr := a.String()
					i, ip, err := net.ParseCIDR(cidr)
					if i.To4() == nil {
						continue
					}

					fmt.Println("\t\t", i, ip, err)
					ip2 := make(net.IP, len(i.To4()))
					binary.BigEndian.PutUint32(ip2, binary.BigEndian.Uint32(ip.IP.To4())|^binary.BigEndian.Uint32(net.IP(ip.Mask).To4()))

					fmt.Println("\t\t", ip2)
					// mask, err := mask(cidr)
					//
					// if err != nil {
					// 	fmt.Println("extracting mask failed:", err)
					// }
					// i, err := mtoi(mask)
					//
					// fmt.Printf("\n %v", net.IP(mask))
					// if err != nil {
					// 	fmt.Println("creating uint16 from mask failed:", err)
					// }
					// fmt.Printf("CIDR: %s\tMask: %d\n", cidr, i)
					// fmt.Println("\t", a)
				}
			}
		}
	}
	fmt.Println("vim-go")
}

// Extracts IP mask from CIDR address.
func mask(cidr string) (net.IPMask, error) {
	_, ip, err := net.ParseCIDR(cidr)
	return ip.Mask, err
}

// Converts IP mask to 16 bit unsigned integer.
func mtoi(mask net.IPMask) (uint16, error) {
	var i uint16
	buf := bytes.NewReader(mask)
	err := binary.Read(buf, binary.BigEndian, &i)
	return i, err
}

func lastAddr(n *net.IPNet) (net.IP, error) { // works when the n is a prefix, otherwise...
	if n.IP.To4() == nil {
		return net.IP{}, errors.New("does not support IPv6 addresses.")
	}
	ip := make(net.IP, len(n.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(n.IP.To4())|^binary.BigEndian.Uint32(net.IP(n.Mask).To4()))
	return ip, nil
}
