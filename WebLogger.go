package main

import (
	"fmt"
	"net"
	"strconv"
	//"time"
)

func main() {
	i_addresses := externalIP()
	for _, i_addr := range i_addresses {
		sendMultiCast(i_addr)
	}

	listenAnswer()
	//	ip := externalIP()
	//	for _, addr := range ip {
	//		fmt.Println(addr.String())
	//	}
}

func sendMultiCast(i_addr string) {
	//	var addr *net.UDPAddr
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.251:5353")
	if err != nil {
		fmt.Printf("Address not resolved!")
		return
	}
	laddr, err := net.ResolveUDPAddr("udp", i_addr+":5353")
	if err != nil {
		fmt.Printf("Local address not resolved!")
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
	requestArray = append(requestArray, 33)
	//	//class
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	//	for {
	fmt.Println("Sending mDNS request from IP: ", i_addr)
	conn, err := net.DialUDP("udp", laddr, addr)
	if err != nil {
		fmt.Println("Dial not sucsesfull!", err.Error())
		return
	}
	defer conn.Close()
	//	for {
	//		fmt.Println(requestArray)
	conn.Write(requestArray)

	//		time.Sleep(10 * time.Second)

	//	}

}

func addStringToArray(str string, requestArray []byte) []byte {
	tmpArray := []byte(str)
	lenStr := byte(len(tmpArray))
	requestArray = append(requestArray, lenStr)
	requestArray = append(requestArray, tmpArray...)
	return requestArray
}

func listenAnswer() {
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.251:5353")
	if err != nil {
		fmt.Printf("Address not resolved!")
		return
	}
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Listen multicast. Dial not sucsesfull!", err.Error())
		return
	}
	defer conn.Close()
	conn.SetReadBuffer(8000)
	for {
		buffer := make([]byte, 8000)
		_, address, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("ReadFromUDP failed:", err)
			//return
		}
		fmt.Println("address: ", address.String())
		parseAnswer(buffer)
	}

}

func parseAnswer(buffer []byte) {
	const base int = 256
	var reqNum int = base*(int)(buffer[4]) + (int)(buffer[5])
	var ansNum int = base*(int)(buffer[6]) + (int)(buffer[7])
	var str string
	var blockBegin int = 12
	var i int
	var ansType int
	var resLen int
	var ip string
	fmt.Println("Parse answer")
	fmt.Println("ansNum: ", ansNum)
	fmt.Println("reqNum: ", reqNum)

	for i = 0; i < reqNum; i++ {
		str, blockBegin = readString(blockBegin, buffer)
		fmt.Println("reg: ", str)
		blockBegin += 4

	}
	if ansNum != 0 {
		str, blockBegin = readString(blockBegin, buffer)
		fmt.Println("Answer string: ", str)
		//		if str!="esp8266._http_._tcp.local"{
		//			continue
		//		}
	}
	for i = 0; i < ansNum; i++ {
		fmt.Println("ansType,blockBegin: ", blockBegin)
		fmt.Println("ansType: ", buffer[blockBegin], ",", buffer[blockBegin+1])
		ansType = base*(int)(buffer[blockBegin]) + (int)(buffer[blockBegin+1])
		blockBegin += 2
		blockBegin += 6
		resLen = base*(int)(buffer[blockBegin]) + (int)(buffer[blockBegin+1])
		fmt.Println("resLen: ", resLen)
		blockBegin += 2
		if ansType == 33 {
			blockBegin += 6
			str, blockBegin = readString(blockBegin, buffer)
			fmt.Println("ansType=33: ", str)
			str, blockBegin = readString(blockBegin, buffer)
			fmt.Println("ansType=33: ", str)

		} else if ansType == 1 {
			fmt.Println("IP: ", buffer[blockBegin], ",", buffer[blockBegin+1], ",", buffer[blockBegin+2], ",", buffer[blockBegin+3])
			ip = strconv.Itoa((int)(buffer[blockBegin])) + "." + strconv.Itoa((int)(buffer[blockBegin+1])) + "." + strconv.Itoa((int)(buffer[blockBegin+2])) + "." + strconv.Itoa((int)(buffer[blockBegin+3]))
			fmt.Println("IP: ", ip)
			return

		} else {
			blockBegin += resLen
		}

	}
}

func readString(reqBegin int, buffer []byte) (string, int) {
	reqStr := ""
	var strLen int
	var i int
	position := reqBegin
	for buffer[position] != 0 {
		if reqStr != "" {
			reqStr += "."
		}
		strLen = (int)(buffer[position])
		for i = 1; i <= strLen; i++ {
			reqStr += string(buffer[position+i])
		}
		position += strLen
		position += 1
	}
	position += 1
	return reqStr, position
}

func externalIP() []string {
	var ipArray []string
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
			for _, v := range ipArray {
				if v == ip.String() {
					continue
				}
			}
			ipArray = append(ipArray, ip.String())
			fmt.Println("Local IP:", ip.String())
		}
	}
	return ipArray
}
