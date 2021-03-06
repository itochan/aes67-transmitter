package sap

import (
	"fmt"
	"log"
	"net"
	"os"
)

const (
	sapAnnounceIP   = "239.255.255.255"
	sapAnnouncePort = 9875
)

type SAP struct {
	interfaceName    string
	HostAddress      net.IP
	MulticastAddress net.IPNet
}

func NewSAP(interfaceName string) *SAP {
	hostAddress := getLocalIPv4Address(interfaceName).IP
	multicastAddress := getUDPMulticastIP(interfaceName)
	return &SAP{
		interfaceName:    interfaceName,
		HostAddress:      hostAddress,
		MulticastAddress: multicastAddress,
	}
}

func (sap *SAP) AnnounceSAP() {
	fmt.Println("Announce SAP...")
	fmt.Printf("Multicast Address: %s:%d\n", sap.MulticastAddress.IP.String(), sapAnnouncePort)

	hostAddress := getLocalIPv4Address(sap.interfaceName).IP
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "AES67 Device"
	}
	destinationAddr := net.UDPAddr{IP: net.ParseIP(sapAnnounceIP), Port: sapAnnouncePort}

	headers := [][]byte{
		{
			0x20,       // Flags
			0x00,       // Authentication Length
			0xff, 0xff, // Message Identifier Hash
		},
		hostAddress.To4(),         // Originating Source
		[]byte("application/sdp"), // Payload Type
		{0x00},                    // Blank
	}
	var header []byte
	for _, h := range headers {
		header = append(header, h...)
	}

	manifest := "v=0\r\n" +
		fmt.Sprintf("o=- 4 0 IN IP4 %s\r\n", hostAddress) +
		fmt.Sprintf("s=%s\r\n", hostname) +
		fmt.Sprintf("c=IN IP4 %s\r\n", sap.MulticastAddress.String()) +
		"t=0 0\r\n" +
		//"a=clock-domain:PTPv2 0\r\n" +
		"m=audio 5004 RTP/AVP 97\r\n" +
		fmt.Sprintf("c=IN IP4 %s\r\n", sap.MulticastAddress.String()) +
		"a=rtpmap:97 L24/48000/2\r\n" +
		"a=sync-time:0\r\n" +
		"a=framecount:48\r\n" +
		"a=ptime:1\r\n" +
		// "a=mediaclk:direct=\r\n" +
		//fmt.Sprintf("a=mediaclk:direct=%d\r\n", time.Now().Unix()) +
		//"a=ts-refclk:ptp=IEEE1588-2008:00-1D-C1-FF-FE-18-E3-24:0\r\n" +
		"a=recvonly\r\n"

	fmt.Println()
	fmt.Print(manifest)

	if err != nil {
		log.Fatal(err)
	}
	dialer := net.Dialer{
		LocalAddr: &net.UDPAddr{IP: getLocalIPv4Address(sap.interfaceName).IP, Port: sapAnnouncePort},
	}
	connect, err := dialer.Dial("udp", destinationAddr.String())
	if err != nil {
		log.Fatal(err)
	}
	defer connect.Close()
	_, err = connect.Write(append(header, []byte(manifest)...))
	if err != nil {
		log.Fatal(err)
	}
}

func getLocalIPv4Address(interfaceName string) *net.IPNet {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		log.Fatal(err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		log.Fatal(err)
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet
			}
		}
	}

	log.Fatalf("Can not get IPv4 address!!")
	return nil
}

func getUDPMulticastIP(interfaceName string) net.IPNet {
	ipAddr := getLocalIPv4Address(interfaceName).IP.To4()
	return net.IPNet{
		IP:   net.IPv4(239, 69, ipAddr[2], ipAddr[3]),
		Mask: net.IPv4Mask(255, 254, 0, 0),
	}
}
