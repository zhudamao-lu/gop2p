package gop2p

import (
	"net"
	"log"
)

const (
	ACTION_CONNECTION_REQUEST = 0x00 // 接入请求
	ACTION_CONNECTION_NOTICE = 0x01 // 新增接入通知

	PACKAGE_HEAD_LEN = 0x08 // 包头固定长度 4 + 4 = 8
	PACKAGE_HEAD_COMMAND_LEN = 0x04 // 包头中命令固定长度 4
	PACKAGE_HEAD_DATA_LEN = 0x04 // 包头中记录数据的固定长度 4
)

func Start(port string) error {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	log.Println("tcp listening on", port, "...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			log.Println("get tcp:", conn.RemoteAddr())

			originData := make([]byte, 0, 65536)
			tcpReadBuffer := make([]byte, 4096)
		//	dataPackage := []byte{}

			for {
				n, err := conn.Read(tcpReadBuffer)
				if err != nil {
					log.Println(err)
					break
				}

				originData = append(originData, tcpReadBuffer[:n]...)
			}
			log.Println(originData, len(originData))

			var i, length, command int

			for i < len(originData) {
				length = int(uint8(originData[i])) << 24 | int(uint8(originData[i + 1])) << 16 | int(uint8(originData[i + 2])) << 8 | int(uint8(originData[i + 3])) // body长度
				command = int(uint8(originData[i + 4])) << 24 | int(uint8(originData[i + 5])) << 16 | int(uint8(originData[i + 6])) << 8 | int(uint8(originData[i + 7])) // 命令号

				if length < 0 {
					log.Println("Error length:", length)
					break
				}

				i += PACKAGE_HEAD_LEN

				data := originData[i : i + length]
				i += length

				log.Println(length, command, i)
				log.Println(string(data))
			}

			log.Println("recieved")
		}()
	}
}

func Decode() {
}
