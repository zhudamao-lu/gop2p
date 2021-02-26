package gop2p

import (
	"net"
	"fmt"
	"log"
)

const head = 0x55
const til = 0xaa

var comingAddrs map[*net.UDPAddr]bool
/*
	action:
	00000000b: 请求接入
	00000001b: 告知接入
	00000010b: 可穿透
*/
type protocol struct {
	rAddr *net.UDPAddr
	action byte
}

func init() {
	comingAddrs = make(map[*net.UDPAddr]bool, 1024)
}

func StartTurnServer(port int) {
	lAddr := &net.UDPAddr{nil, port, ""}
	ln, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Println(ln.LocalAddr())

	for {
		p := make([]byte, 65505)
		n, rAddr, err := ln.ReadFromUDP(p)
		if err != nil {
			log.Println(err)
			break
		}

		go handler(ln, rAddr, p[:n])
	}
}

func handler(ln *net.UDPConn, rAddr *net.UDPAddr, p []byte) {
	log.Println(rAddr)
	l := len(p)
	fmt.Printf("%08b, %v, %08b\n", p[0], p[1:l - 1], p[l - 1])

	if p[0] != 0x55 || p[len(p) - 1] != 0xaa {
		log.Println("invalid udp packet coming")
		return
	}

	pc := protocol{rAddr, p[1]}
	data := p[2:l - 1]

	switch pc.action {
	case 0:
		if len(comingAddrs) == 0 {
			comingAddrs[rAddr] = true
			break
		}
		sendData := []byte{head}
		sendData = append(sendData, byte(1))
		for k, _ := range comingAddrs {
			sendData = append(sendData, []byte(k.IP)...)
			portH := k.Port >> 8
			portL := k.Port << 8 >> 8
			sendData = append(sendData, []byte{byte(portH), byte(portL)}...)
		}
		sendData = append(sendData, til)

		/*
		l := len(sendData)
		fmt.Printf("%08b, %v, %08b\n", sendData[0], sendData[1:l - 1], sendData[l - 1])
		*/

		if len(sendData) > 65505 {
			log.Fatal("Max data length is 65505")
		}

		n, err := ln.WriteToUDP(sendData, rAddr)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

		comingAddrs[rAddr] = true
		log.Println(string(data))
		log.Println(comingAddrs)
	case 1:
		log.Println(data)
		ip := net.IP(data[:16])
		sAddr := &net.UDPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(sAddr)
	}
}
