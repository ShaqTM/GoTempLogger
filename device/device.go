package device

import (
	"Weblogger/mdatabase"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

type Device struct {
	Name            string
	RefreshInterval time.Duration
	Ip              string
}

func (device Device) ReadFromDevice(mdb mdatabase.MDB) {
	for {
		for device.Ip == "" {
			device = device.findDevice()
			if device.Ip == "" {
				time.Sleep(time.Second * 10)
			}
		}

		timeout := time.Duration(10 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get("http://" + device.Ip + "/tempData")
		if err != nil {
			device = device.findDevice()
			if device.Ip != "" {
				resp, err = client.Get("http://" + device.Ip + "/tempData")
			}
		}
		if err == nil {
			responseText, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				fmt.Println("Response: ", string(responseText))
				mdb.Insert_data(responseText, device.Name)
			}
		}
		time.Sleep(device.RefreshInterval * time.Second)
	}
}

func (device Device) findDevice() Device {

	requestArray := device.buildRequest()
	ifaces := externalIP()
	for _, iface := range ifaces {
		device := device.sendMultiCast(iface, requestArray)
		if device.Ip != "" {
			return device
		}
	}
	return device

}

func (device Device) sendMultiCast(iface net.Interface, requestArray []byte) Device {
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.251:5353")
	if err != nil {
		fmt.Printf("Address not resolved!", err.Error())
		return device
	}
	i_addr := getIP(iface)
	_, err = net.ResolveUDPAddr("udp", i_addr+":5353")
	if err != nil {
		fmt.Printf("Local address not resolved!", err.Error())
		return device
	}
	connOpened := false
	conn, err := net.ListenMulticastUDP("udp", &iface, addr)
	if err == nil {
		connOpened = true
	}
	for connOpened == false {
		conn, err = net.ListenMulticastUDP("udp", &iface, addr)
		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		connOpened = true
	}
	defer conn.Close()
	conn.SetReadBuffer(8000)

	timeout := true
	for i := 0; i < 10; i++ {
		if timeout {
			_, err := conn.WriteToUDP(requestArray, addr)
			if err != nil {
				fmt.Println("WriteToUDP not sucsesfull!", err.Error())
				return device
			}
		}
		timeout = false
		buffer := make([]byte, 8000)

		err = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			fmt.Println("SetReadDeadLine not sucsesfull!", err.Error())
			return device
		}
		_, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			e, _ := err.(net.Error)
			if e.Timeout() {
				timeout = true
				continue
			}
			fmt.Println("ReadFromUDP failed:", err.Error())
			return device
		}

		device := device.parseAnswer(buffer)
		if device.Ip != "" {
			fmt.Println("Found device with IP = ", device.Ip)
			return device
		}

	}
	return device

}

func (device Device) buildRequest() []byte {
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
	requestArray = addStringToArray(device.Name, requestArray)
	requestArray = addStringToArray("_http", requestArray)
	requestArray = addStringToArray("_tcp", requestArray)
	requestArray = addStringToArray("local", requestArray)
	requestArray = append(requestArray, 0)

	//	//type
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 33)
	//	//class
	requestArray = append(requestArray, 0)
	requestArray = append(requestArray, 1)
	return requestArray

}

func addStringToArray(str string, requestArray []byte) []byte {
	tmpArray := []byte(str)
	lenStr := byte(len(tmpArray))
	requestArray = append(requestArray, lenStr)
	requestArray = append(requestArray, tmpArray...)
	return requestArray
}

func (device Device) parseAnswer(buffer []byte) Device {
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
		fmt.Println("Answer string: '", str+"'")
		if str != device.Name+"._http._tcp.local" {
			return device
		}
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
			device.Ip = ip
			return device

		} else {
			blockBegin += resLen
		}

	}
	return device
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

func externalIP() []net.Interface {
	var ifArray []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return ifArray
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		_, err := iface.Addrs()
		if err != nil {
			continue
		}
		ifArray = append(ifArray, iface)
	}
	return ifArray
}

func getIP(iface net.Interface) string {
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue // not an ipv4 address
		}
		return ip.String()
		fmt.Println("Local IP:", ip.String())
	}
	return ""

}
