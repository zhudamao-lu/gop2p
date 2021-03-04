package gop2p

import (
	"errors"
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

/*
func tcpTurnBySeedAddr(conn *net.TCPConn, lAddr, rAddr *net.TCPAddr) {
}
*/

func StartTCPTurnServer() error {
	lAddr := &net.TCPAddr{nil, 1024, ""}
	ln, err := net.ListenTCP("tcp", lAddr)
	if err != nil {
		return err
	}
	log.Println(ln.Addr())
	defer ln.Close()

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(conn.RemoteAddr())

		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn *net.TCPConn) {
	defer conn.Close()
	err := conn.SetNoDelay(true)
	if err != nil {
		log.Println(err)
	}

	data := make([]byte, 0, 4096)
	var bodyLength, command int
	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			log.Println(err)
			break
		}

		log.Println(n, buffer[:n])
		data = append(data, buffer[:n]...)

		LABEL:
		log.Println(string(data[:PACKET_IDENTIFY_LEN]))
		if string(data[:PACKET_IDENTIFY_LEN]) == PACKET_IDENTIFY {
			command, bodyLength, err = DecodeData(data)
			if err != nil {
				log.Println(err)
				break
			}
		}

		log.Println(bodyLength + PACKET_HEAD_LEN, len(data))
		bodyEnd := bodyLength + PACKET_HEAD_LEN
		if bodyEnd <= len(data) {
			tcpHandle(command, data[PACKET_HEAD_LEN : bodyEnd])
			mod := bodyEnd % 1024
			data = buffer[mod : n]
			buffer = data[:]
			n -= mod
			goto LABEL
		}
	//	log.Println(string(data[:PACKET_IDENTIFY_LEN]))
	}

	log.Println(data)
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
	log.Println(command, bodyLength)

	return command, bodyLength, nil
}

func tcpHandle(command int, data []byte) {
	switch command {
	case 0xffff:
		log.Println(string(data))
	}
}

func bytesToInt(b []byte) (int, error) {
	if len(b) != 4 {
		return 0, errors.New("bytes length must be 4")
	}

	return int(uint8(b[0])) << 24 | int(uint8(b[1])) << 16 | int(uint8(b[2])) << 8 | int(uint8(b[3])), nil
}
