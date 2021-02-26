package main

import (
	"net"
	"fmt"
	"log"
)

const head = 0x55
const til = 0xaa

var seedAddrs = make([]*net.UDPAddr, 0, 1024)

/*
	action:
	00000000b: 请求接入
	00000001b: 告知接入
	00000010b: 可穿透
*/
func init() {
	addr, err := net.ResolveUDPAddr("udp4", "121.41.85.45:1024")
	if err != nil {
		log.Println(err)
	}

	seedAddrs = append(seedAddrs, addr)
}

func main() {
	var err error
	for _, v := range seedAddrs {
		err = dialUDP(v)
		if err != nil {
			log.Println(err)
		}
	}
}

func dialUDP(rAddr *net.UDPAddr) error {
	ln, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		return err
	}
	//	defer ln.Close()

	go func() {
		data := []byte{head}
		data = append(data, 0)
		data = append(data, []byte("hello server")...)
		data = append(data, til)

		if len(data) > 65505 {
			log.Fatal("Max data length is 65505")
		}

		ln.Write(data)
	}()

	for {
		p := make([]byte, 65505)
		n, rAddr, err := ln.ReadFromUDP(p)
		if err != nil {
			return err
		}

		go handler(rAddr, p[:n])
	}
}

func handler(rAddr *net.UDPAddr, p []byte) {
	log.Println(rAddr)
	l := len(p)
	fmt.Printf("%08b, %v, %08b\n", p[0], p[1:l - 1], p[l - 1])

	if p[0] != 0x55 || p[len(p) - 1] != 0xaa {
		log.Println("invalid udp packet coming")
		return
	}

	data := p[2:l - 1]

	switch p[1] {
	case 0:
		/*
		for k, _ := range comingAddrs {
			ln, err := net.DialUDP("udp", nil, k)
			if err != nil {
				log.Fatal(err)
			}
		//	defer ln.Close()

			data := []byte{head}
			data = append(data, 1)
			data = append(data, []byte(k.IP)...)
			portH := k.Port / 0x100
			portL := k.Port % 0x100
			data = append(data, []byte{byte(portH), byte(portL)}...)
			data = append(data, til)

			if len(data) > 65505 {
				log.Fatal("Max data length is 65505")
			}

			ln.Write(data)
		}
		comingAddrs[rAddr] = true
		log.Println(string(data))
		*/
	case 1:
		log.Println(data)
		ip := net.IP(data[:16])
		sAddr := &net.UDPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(sAddr)
	}
}
