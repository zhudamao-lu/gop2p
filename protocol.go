package gop2p

import (
	"runtime"
	"context"
	"net"
	"log"
)

const (
	ACTION_CONNECTION_REQUEST = 0x00 // 接入请求
	ACTION_CONNECTION_NOTICE = 0x01 // 新增接入通知

	PACKET_IDENTIFY = "--TCPHEADX--" // 包开始标识符
	PACKET_IDENTIFY_LEN = 0xc // 包开始标识符长度 12
	PACKET_COMMAND_LEN = 0x4 // 包头中命令固定长度 4
	PACKET_COMMAND_END_LEN = 0x10 // 包头中开始到命令结束固定长度 12 + 4
	PACKET_BODY_SIZE_LEN = 0x4 // 包头中记录数据的固定长度 4
	PACKET_HEAD_LEN = 0x14 // 包头固定长度 12 + 4 + 4 = 20
)

var comingAddrs map[*net.TCPAddr]bool
var seedAddrs map[*net.TCPAddr]bool

/*
func tcpTurnBySeedAddr(conn *net.TCPConn, lAddr, rAddr *net.TCPAddr) {
}
*/

func ConnectSeed() error {
	seedAddrs = make(map[*net.TCPAddr]bool, 1024)
	addr, err := net.ResolveTCPAddr("tcp", "121.41.85.45:43299")
	if err != nil {
		return err
	}

	seedAddrs[addr] = true

	return nil
}

func StartTCPTurnServer() error {
	log.Println(runtime.GOOS)
	var listenConfig net.ListenConfig
	switch runtime.GOOS {
	case "linux":
		listenConfig = net.ListenConfig{Control: controlSockReusePortUnix}
		/*
	case "windows":
		listenConfig = net.ListenConfig{Control: controlSockReusePortWindow}
		*/
	}

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

	ConnectSeed()

	for k, _ := range seedAddrs {
		log.Println(k)

		d := net.Dialer {Control: controlSockReusePortUnix, LocalAddr: lAddr}
		connc, err := d.Dial(k.Network(), k.String())
		if err != nil {
			log.Println(err)
			continue
		}

		body := []byte("hello server")
		data := []byte(PACKET_IDENTIFY)
		data = append(data, intToBytes(0xffffffff)...)
		data = append(data, intToBytes(len(body))...)
		data = append(data, body...)
		log.Println(data)

		n, err := connc.Write(data)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(n)
	}

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

			if PACKET_HEAD_LEN + bodyLength <= len(data) {
				body := data[PACKET_HEAD_LEN : PACKET_HEAD_LEN + bodyLength]
				tcpHandle(command, body)
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

func tcpHandle(command int, data []byte) {
	switch command {
	case ACTION_CONNECTION_REQUEST:
		/*
			穿透服收到接入请求
			连接关系:
			source:B distination:S
		*/

	case 0xffffffff: // 用于测试TCP
		log.Println(string(data))
	}
}
