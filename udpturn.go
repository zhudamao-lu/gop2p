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

	/*
	addr, err := net.ResolveUDPAddr("udp4", "121.41.85.45:52475")
	if err != nil {
		log.Println(err)
	}

	seedAddrs[addr] = true
	*/

}

func udpTurnBySeedAddr(conn *net.UDPConn, lAddr, rAddr *net.UDPAddr) {
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

		go handlerUDPConnection(conn, rAddr, p[:n])
	}
}

func StartUDPTurnServer() error {
	lAddr := &net.UDPAddr{nil, 0, ""} // 本机服务端口
	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("localAddr:", conn.LocalAddr())

	for k, _ := range seedAddrs {
		go udpTurnBySeedAddr(conn, lAddr, k)
	}

	readFromUDP(conn)

	return nil
}

func handlerUDPConnection(conn *net.UDPConn, rAddr *net.UDPAddr , p []byte) {
	log.Println("remoteAddr:", rAddr)
	l := len(p)
	fmt.Printf("%08b, %v, %08b\n", p[0], p[1:l - 1], p[l - 1])

	if p[0] != head || p[len(p) - 1] != til {
		log.Println("invalid udp packet coming")
		return
	}

//	pc := protocol{rAddr, p[1]}
	data := p[2:l - 1]

	// 以下注释连接关系 新近客户端为:A 其他客户端为:B 穿透服为:S
//	switch pc.action {
	switch p[1] {
	case 0:
		/*
			穿透服收到接入请求
			连接关系:
			source:B distination:S
		*/
		log.Println("case 0:", data)
		log.Println(comingAddrs)

		// 穿透服告知之前已接入的节点，有新节点加入
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
		/*
			收到穿透服务器发来的，新节点加入信息
			对他进行访问用来打洞，并告知穿透服这一操作
			连接关系:
			source:S distination:A data:B
		*/
		log.Println("case 1:", data)

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.UDPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(rAddrC)

		// 向对方客户端发信息打洞
		lAddr, err := net.ResolveUDPAddr("udp4", conn.LocalAddr().String())
		if err != nil {
			log.Println(err)
		}

		_, err = net.DialUDP("udp4", lAddr, rAddrC)
		if err != nil {
			log.Println(err)
		}

		sendData := []byte{head}
		sendData = append(sendData, byte(2))
		sendData = append(sendData, []byte("turn")...)
		sendData = append(sendData, til)
		n, err := conn.WriteToUDP(sendData, rAddrC)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

		// 告知穿透服已打洞
		sendData = []byte{head}
		sendData = append(sendData, byte(3))
		sendData = append(sendData, []byte("turned")...)
		sendData = append(sendData, data...)
		sendData = append(sendData, til)
		n, err = conn.WriteToUDP(sendData, rAddr)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)
	case 2:
		/*
			本次消息由于网络环境可能被忽略
			第一次
			如第一次被忽略
			但是可以帮助打洞
			连接关系:
			source:A distination:B 

			如未被忽略则第二次会被忽略
			第二次
			连接关系:
			source:B distination:A

			其中有一次未被忽略的将会收到"hello,brother"消息
			表示穿透成功，可以在此case中继续后续回调执行上层逻辑
		*/
		log.Println("case 2:", string(data))
	case 3:
		/*
			穿透服务收到打洞回馈信息
			连接关系:
			source:A distination:S data:B
		*/
		log.Println("case 3:", data)
		log.Println(string(data[:6]))

		// Decode 对方客户端 addr
		ip := net.IP(data[6:22])
		rAddrC := &net.UDPAddr{ip, int(data[22]) << 8 | int(data[23]), ""}
		log.Println(rAddrC)

		// 封装数据 Encode 另一客户端地址为data 
		lAddr, err := net.ResolveUDPAddr("udp4", conn.LocalAddr().String())
		if err != nil {
			log.Println(err)
		}

		_, err = net.DialUDP("udp4", lAddr, rAddrC)
		if err != nil {
			log.Println(err)
		}

		sendData := []byte{head}
		sendData = append(sendData, byte(4))
		sendData = append(sendData, rAddr.IP...)
		portH := rAddr.Port >> 8
		portL := rAddr.Port << 8 >> 8
		log.Println(rAddr.IP, rAddr.Port)
		sendData = append(sendData, uint8(portH), uint8(portL))
		sendData = append(sendData, til)
		n, err := conn.WriteToUDP(sendData, rAddrC)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)
	case 4:
		/*
			客户端收到穿透服告知其他客户端已向自己打洞
			开始向其他客户端发送消息
			连接关系:
			source:S distination:B data:A
		*/
		log.Println("case 4:", data)

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.UDPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(rAddrC)

		// 封装数据 发送 "hello,brother"
		sendData := []byte{head}
		sendData = append(sendData, byte(2))
		sendData = append(sendData, []byte("hello,brother")...)
		sendData = append(sendData, til)
		n, err := conn.WriteToUDP(sendData, rAddrC)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)
	case 0xff:
		log.Println(string(data))
	}
}
