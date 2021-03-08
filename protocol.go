package gop2p

import (
	"runtime"
	"context"
	"net"
	"log"
)

const (
	ACTION_CONNECTION_REQUEST = 0x00 // 接入请求
	ACTION_CONNECTION_NOTICE = 0x01 // 新增接入通知，并开始穿透
	ACTION_CONNECTION_TURN = 0x02 // 收到则穿透成功
	ACTION_CONNECTION_TURNED = 0x03 // 服务器被告知已做过一次穿透，并通知另一客户端反向访问
	ACTION_CONNECTION_NOTICE2 = 0x04 // 第二次穿透访问

	PACKET_IDENTIFY = "--TCPHEADX--" // 包开始标识符
	PACKET_IDENTIFY_LEN = 0xc // 包开始标识符长度 12
	PACKET_COMMAND_LEN = 0x4 // 包头中命令固定长度 4
	PACKET_COMMAND_END_LEN = 0x10 // 包头中开始到命令结束固定长度 12 + 4
	PACKET_BODY_SIZE_LEN = 0x4 // 包头中记录数据的固定长度 4
	PACKET_HEAD_LEN = 0x14 // 包头固定长度 12 + 4 + 4 = 20
)

var (
	seedAddrs = make(map[*net.TCPAddr]bool, 1024)
	comingConns = make(map[*net.TCPConn]bool, 1024)
	peers = make(map[*net.TCPConn]bool, 1024)
)

func ConnectSeed(lAddr *net.TCPAddr, seedAddrsStr []string) error {
	for _, v := range seedAddrsStr {
		addr, err := net.ResolveTCPAddr("tcp", v)
		if err != nil {
			return err
		}

		seedAddrs[addr] = true
	}

	for k, _ := range seedAddrs {
		log.Println(k)

		d := net.Dialer {Control: controlSockReusePortUnix, LocalAddr: lAddr}
	//	d := net.Dialer {Control: controlSockReusePortWindow, LocalAddr: lAddr}
		connc, err := d.Dial(k.Network(), k.String())
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(connc.LocalAddr())

		body := []byte("hello server")
		data := []byte(PACKET_IDENTIFY)
		data = append(data, intToBytes(ACTION_CONNECTION_REQUEST)...)
		data = append(data, intToBytes(len(body))...)
		data = append(data, body...)
		log.Println(data)

		go handleTCPConnection(connc.(*net.TCPConn))
		n, err := connc.Write(data)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(n)
	}

	return nil
}

func StartTCPTurnServer(seedAddrsStr []string) error {
	log.Println(runtime.GOOS)
	var listenConfig net.ListenConfig
	listenConfig = net.ListenConfig{Control: controlSockReusePortUnix}
//	listenConfig = net.ListenConfig{Control: controlSockReusePortWindow}

//	lAddr := &net.TCPAddr{nil, 1024, ""}
	ln, err := listenConfig.Listen(context.Background(), "tcp", "")
	if err != nil {
		return err
	}
	log.Println(ln.Addr())

	lAddr, err := net.ResolveTCPAddr(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		return err
	}
	log.Println(lAddr)

	ConnectSeed(lAddr, seedAddrsStr)

	listenAccept(ln)

	return nil
}

func handleTCPConnection(conn *net.TCPConn) {
	defer conn.Close()

	data := make([]byte, 0, 4096)
	var bodyLength, command int
	for {
		buffer := make([]byte, 64)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
			break
		}

	//	log.Println(n, buffer[:n])
		data = append(data, buffer[:n]...)
	//	log.Println(string(data))

		for string(data[:PACKET_IDENTIFY_LEN]) == PACKET_IDENTIFY {
			command, bodyLength, err = DecodeData(data)
			if err != nil {
				log.Println(err)
				break
			}

			bodyEnd := PACKET_HEAD_LEN + bodyLength
			if bodyEnd  <= len(data) {
				body := data[PACKET_HEAD_LEN : bodyEnd]
				tcpHandle(command, body, conn)
				data = data[PACKET_HEAD_LEN + bodyLength :]
				continue
			}

			break
		}
	}

//	log.Println(data)
}

func listenAccept(ln net.Listener) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(conn.RemoteAddr())

		go handleTCPConnection(conn.(*net.TCPConn))
	}
}

func DecodeData(data []byte) (int, int, error) {
	command, err := bytesToInt(data[PACKET_IDENTIFY_LEN : PACKET_COMMAND_END_LEN])
	if err != nil {
		data = data[0:0]
		return 0, 0, err
	}
	bodyLength, err := bytesToInt(data[PACKET_COMMAND_END_LEN : PACKET_HEAD_LEN])
	if err != nil {
		data = data[0:0]
		return 0, 0, err
	}
//	log.Println(command, bodyLength)

	return command, bodyLength, nil
}

func tcpHandle(command int, data []byte, conn *net.TCPConn) {
	switch command {
	/*
		穿透服收到接入请求
		连接关系:
		source:B distination:S
	*/
	case ACTION_CONNECTION_REQUEST:
		log.Println("case 0:", data)
		for k, v := range comingConns {
			log.Println(k.LocalAddr(), k.RemoteAddr(), v)
		}

		rAddr, err := net.ResolveTCPAddr("tcp", conn.RemoteAddr().String())
		if err != nil {
			log.Println(err)
			break
		}

		// 穿透服告知之前已接入的节点，有新节点加入
		for k, _ := range comingConns {
			body := []byte(rAddr.IP)
			portH := rAddr.Port >> 8
			portL := rAddr.Port << 8 >> 8
			body = append(body, uint8(portH), uint8(portL))
			sendData := []byte(PACKET_IDENTIFY)
			sendData = append(sendData, intToBytes(ACTION_CONNECTION_NOTICE)...)
			sendData = append(sendData, intToBytes(len(body))...)
			sendData = append(sendData, body...)
			log.Println("sendData:", sendData)
			log.Println("k:", k.RemoteAddr())
			n, err := k.Write(sendData)
			if err != nil {
				log.Println(err, "xxxxxxxxxxxxxxxxx")
				delete(comingConns, k)
				continue
			}
			log.Println("n:", n)
		}

		comingConns[conn] = true
		log.Println(string(data))

	/*
		收到穿透服务器发来的，新节点加入信息
		对他进行访问用来打洞，并告知穿透服这一操作
		连接关系:
		source:S distination:A data:B
	*/
	case ACTION_CONNECTION_NOTICE:
		log.Println("case 1:", data)

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.TCPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(rAddrC)

		// 向对方客户端发信息打洞
		lAddr, err := net.ResolveTCPAddr(conn.LocalAddr().Network(), conn.LocalAddr().String())
		if err != nil {
			log.Println(err)
		}

		d := net.Dialer {Control: controlSockReusePortUnix, LocalAddr: lAddr}
		connc, err := d.Dial(rAddrC.Network(), rAddrC.String())
		if err != nil {
			log.Println(err)
		}

		body := []byte("turn...")
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURN)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		log.Println("sendData:", sendData)
		n, err := connc.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

		// 告知穿透服已打洞
		body = []byte("turned...")
		body = append(body, data...)
		sendData = []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURNED)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err = conn.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

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
	case ACTION_CONNECTION_TURN:
		log.Println("case 2:", data)
		log.Println(string(data[:7]))
		for k, _ := range comingConns {
			if k.RemoteAddr() == conn.RemoteAddr() {
				return
			}
		}

		log.Println("节点连接成功")
		peers[conn] = true

	/*
		穿透服务收到打洞回馈信息
		连接关系:
		source:A distination:S data:B
	*/
	case ACTION_CONNECTION_TURNED:
		log.Println("case 3:", data)
		log.Println(string(data[:9]))

		// Decode 对方客户端 addr
		ip := net.IP(data[9:25])
		rAddrC := &net.TCPAddr{ip, int(data[25]) << 8 | int(data[26]), ""}
		log.Println(rAddrC)

		rAddr, err := net.ResolveTCPAddr(conn.LocalAddr().Network(), conn.RemoteAddr().String())
		if err != nil {
			log.Println(err)
			break
		}

		var connS *net.TCPConn
		for k, _ := range comingConns {
			if k.RemoteAddr().String() == rAddrC.String() {
				connS = k
				break
			}
		}

		// 封装数据 Encode 另一客户端地址为 body
		body := []byte(rAddr.IP)
		body = append(body, uint8(rAddr.Port >> 8), uint8(rAddr.Port << 8 >> 8))
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_NOTICE2)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err := connS.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

	/*
		客户端收到穿透服告知其他客户端已向自己打洞
		开始向其他客户端发送消息
		连接关系:
		source:S distination:B data:A
	*/
	case ACTION_CONNECTION_NOTICE2:
		log.Println("case 4:", data)

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.TCPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		log.Println(rAddrC)

		// 封装数据 发送 "turn..."
		body := []byte("turn...")
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURN)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err := conn.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

//	case ACTION_CONNECTION_API:


	/*
		用于测试TCP
	*/
	case 0xffffffff:
		log.Println(string(data))
	}
}
