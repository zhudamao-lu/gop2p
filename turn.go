package gop2p

import (
	"net"
	"fmt"
	"log"
)

const head = 0x55
const til = 0xaa

var comingAddrs map[*net.UDPAddr]bool
var seedAddrs map[*net.UDPAddr]bool
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
	seedAddrs = make(map[*net.UDPAddr]bool, 1024)

	addr, err := net.ResolveUDPAddr("udp4", "121.41.85.45:32804")
	if err != nil {
		log.Println(err)
	}

	seedAddrs[addr] = true

}

func turnBySeedAddr(conn *net.UDPConn, lAddr, rAddr *net.UDPAddr) {
	data := []byte{head}
	data = append(data, 0)
	data = append(data, []byte("hello server")...)
	data = append(data, til)

	if len(data) > 65505 {
		log.Fatal("Max data length is 65505")
	}

	conn.WriteToUDP(data, rAddr)
}

func readFromUDP(conn *net.UDPConn) {
	for {
		p := make([]byte, 65505)
		n, rAddr, err := conn.ReadFromUDP(p)
		if err != nil {
			log.Println(err)
			break
		}

		go handler(conn, rAddr, p[:n])
	}
}

func StartTurnServer() error {
	lAddr := &net.UDPAddr{nil, 0, ""} // 本机服务端口
	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("localAddr:", conn.LocalAddr())

	for k, _ := range seedAddrs {
		go turnBySeedAddr(conn, lAddr, k)
	}

	readFromUDP(conn)

	return nil
}

func handler(conn *net.UDPConn, rAddr *net.UDPAddr , p []byte) {
	log.Println("remoteAddr:", rAddr)
	l := len(p)
	fmt.Printf("%08b, %v, %08b\n", p[0], p[1:l - 1], p[l - 1])

	if p[0] != head || p[len(p) - 1] != til {
		log.Println("invalid udp packet coming")
		return
	}

//	pc := protocol{rAddr, p[1]}
	data := p[2:l - 1]

//	switch pc.action {
	switch p[1] {
	case 0:
		log.Println(comingAddrs)
		for k, _ := range comingAddrs {
			sendData := []byte{head}
			sendData = append(sendData, byte(1))
			sendData = append(sendData, rAddr.IP...)
			portH := rAddr.Port >> 8
			portL := rAddr.Port << 8 >> 8
			log.Println(rAddr.IP, rAddr.Port)
			sendData = append(sendData, uint8(portH), uint8(portL))
			sendData = append(sendData, til)
			log.Println("sendData:", sendData)
			log.Println("k:", k)
			n, err := conn.WriteToUDP(sendData, k)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println("n:", n)
		}

		comingAddrs[rAddr] = true
		log.Println(string(data))
	//	log.Println(comingAddrs)
	case 1:
		log.Println("case 1", data)
		log.Println(conn.RemoteAddr())
		ip := net.IP(data[:16])
		rAddr := &net.UDPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(rAddr)

		sendData := []byte{head}
		sendData = append(sendData, byte(2))
		sendData = append(sendData, []byte("turn")...)
		sendData = append(sendData, til)
		n, err := conn.WriteToUDP(data, rAddr)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)
		log.Println(conn.RemoteAddr())
	case 2:
		log.Println(string(data))
	case 0xff:
		log.Println(string(data))
	}
}
