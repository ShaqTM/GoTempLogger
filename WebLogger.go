package main

import (
	"fmt"
	"net"
)

func main() {
	//go sendMultiCast()
	ip := externalIP()
	for _, addr := range ip {
		fmt.Printf(addr.String())
	}
}

func sendMultiCast() {
	//	var addr *net.UDPAddr
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.251:5353")
	if err != nil {
		fmt.Printf("Address not resolved!")
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("Dial not sucsesfull!")
		return
	}
	var requestArray []byte
	//ID
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//Flags
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of questions
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	//number of answers
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of ///
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//number of ///
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//device name
	//writeName("esp8266._http._tcp.local", message)
	requestArray = addStringToArray("esp8266", requestArray)
	requestArray = addStringToArray("_http_", requestArray)
	requestArray = addStringToArray("_tcp", requestArray)
	requestArray = addStringToArray("local", requestArray)
	requestArray = append(requestArray, 0)

	//	//type
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 0)
	//	//class
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	conn.Write(requestArray)
}

func addStringToArray(str string, requestArray []byte) []byte {
	tmpArray := []byte(str)
	len := byte(cap(tmpArray))
	requestArray = append(requestArray, len)
	requestArray = append(requestArray, tmpArray...)
	return requestArray
}

func listenAnswer() {

}

func externalIP() []net.IP {
	var ipArray []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return ipArray
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ipArray = append(ipArray, ip)
			fmt.Println(ip.String())
		}
	}
	return ipArray
}
