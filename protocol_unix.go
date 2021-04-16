//	+build !windows

/*
	Copyright 2021 Mosalut and (此处将为公司名称). All rights reserved.
	Use of this source code is governed by a BSD-style
	license that can be found in the LICENSE file.

	github.com/mosalut/gop2p包，提供了一组底层的TCP peer to peer网络交互功能
	主要包括穿透、新增节点通知、节点之间交互等功能

	通常，你只需要调用 StartTCPTurnServer 方法即可新建或加入到一个peer to peer网络
*/
package gop2p

import (
	"context"
	"net"
	"log"
)

const (
	ACTION_CONNECTION_REQUEST = 0x00 // 接入请求
	ACTION_CONNECTION_NOTICE = 0x01 // 新增接入通知，并开始穿透
	ACTION_CONNECTION_TURN_OK = 0x02 // 收到则穿透成功
	ACTION_CONNECTION_TURNING = 0x03 // 服务器被告知已做过一次穿透，并通知另一客户端反向访问
	ACTION_CONNECTION_NOTICE2 = 0x04 // 第二次穿透访问
	ACTION_CONNECTION_LOGIC = 0x05 // peer to peer 之间处理正式通信内容

	PACKET_IDENTIFY = "--TCPHEADX--" // 包开始标识符
	PACKET_IDENTIFY_LEN = 0xc // 包开始标识符长度 12
	PACKET_COMMAND_LEN = 0x4 // 包头中命令固定长度 4
	PACKET_COMMAND_END_LEN = 0x10 // 包头中开始到命令结束固定长度 12 + 4
	PACKET_BODY_SIZE_LEN = 0x4 // 包头中记录数据的固定长度 4
	PACKET_HEAD_LEN = 0x14 // 包头固定长度 12 + 4 + 4 = 20
)

var (
	seedAddrs = make(map[*net.TCPAddr]bool, 1024) // 种子节点地址map
	comingConns = make(map[*net.TCPConn]bool, 1024) // 自身看作穿透服时，新连接map
	peers = make(map[*net.TCPConn]bool, 1024) // 自身看作客户端时，新连接map

//	EventsArrayFunc [5]func(args ...interface{}) error // 回调事件函数指针数组
)

type Event_T struct {
	Args [5]interface{}
	OnRequest func(command int) error
	OnNotice func(command int) error
	OnOK func(command int) error
	OnTurning func(command int) error
	OnNotice2 func(command int) error
}

func GetPeers() map[*net.TCPConn]bool {
	return peers
}

func GetSeedAddrs() map[*net.TCPAddr]bool {
	return seedAddrs
}

/*
	连接种子节点
*/

// func connectSeed(lAddr *net.TCPAddr, seedAddrsStr []string, processLogic func(int, []byte, *net.TCPConn) error) error {
func connectSeed(lAddr *net.TCPAddr, seedAddrsStr []string, event *Event_T, processLogic func(int, []byte, *net.TCPConn) error) error {
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

		go handleTCPConnection(connc.(*net.TCPConn), event, processLogic)
		n, err := connc.Write(data)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(n)
	}

	return nil
}

/*
	StartTCPTurnServer 返回一个 error.
	参数 seedAddrsStr 接收一个字符串列表作为一批种子地址,
	其中的任意一个如果转换为 *net.TCPAddr 失败, 都会返回一个 error,
	并且中断后续执行.

	该 func 会设置端口重用, 在同一端口监听和拨号. 当发生监听、拨号类问题,
	也会返回一个 error.

	参数 processLogic 接收一个 func, 该 func 是在穿透成功建立点对点连接,
	并且收到对方节点传来的数据时，根据起第一个参数(int 类型，可看作API号),
	做出的相应动作. 第一个参数将会是对方传来数据的body中前4个字节.
	该func 由用户实现, 并传入 StartTCPTurnServer.
*/
func StartTCPTurnServer(seedAddrsStr []string, event *Event_T, processLogic func(int, []byte, *net.TCPConn) error) error {
	var listenConfig net.ListenConfig
	listenConfig = net.ListenConfig{Control: controlSockReusePortUnix}

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

//	connectSeed(lAddr, seedAddrsStr, eventsArrayFunc, processLogic)
	connectSeed(lAddr, seedAddrsStr, event, processLogic)

//	listenAccept(ln, eventsArrayFunc, processLogic)
	listenAccept(ln, event, processLogic)

	return nil
}

func handleTCPConnection(conn *net.TCPConn, event *Event_T, processLogic func(int, []byte, *net.TCPConn) error) {
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

		data = append(data, buffer[:n]...)

		for string(data[:PACKET_IDENTIFY_LEN]) == PACKET_IDENTIFY {
			command, bodyLength, err = decodeData(data)
			if err != nil {
				log.Println(err)
				break
			}

			bodyEnd := PACKET_HEAD_LEN + bodyLength
			if bodyEnd  <= len(data) {
				body := data[PACKET_HEAD_LEN : bodyEnd]
				tcpHandle(command, body, conn, event, processLogic)
				data = data[PACKET_HEAD_LEN + bodyLength :]
				continue
			}

			break
		}
	}
}

func listenAccept(ln net.Listener, event *Event_T, processLogic func(int, []byte, *net.TCPConn) error) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(conn.RemoteAddr())

		go handleTCPConnection(conn.(*net.TCPConn), event, processLogic)
	}
}

func decodeData(data []byte) (int, int, error) {
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

	return command, bodyLength, nil
}

func tcpHandle(command int, data []byte, conn *net.TCPConn, event *Event_T, processLogic func(int, []byte, *net.TCPConn) error) {
	defer handlePanic("tcpHandle")

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
				delete(comingConns, k)
				continue
			}
			log.Println("n:", n)
		}

		comingConns[conn] = true
		log.Println(string(data))

		err = event.OnRequest(command)
		if err != nil {
			log.Println(err)
			break
		}

		/*
		if EventsArrayFunc[command] == nil { break }
		err = EventsArrayFunc[command]()
		if err != nil {
			log.Println(err)
			break
		}
		*/



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

		go handleTCPConnection(connc.(*net.TCPConn), event, processLogic)

		body := []byte("turn...")
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURN_OK)...)
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
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURNING)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)
		n, err = conn.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

		err = event.OnNotice(command)
		if err != nil {
			log.Println(err)
			break
		}

		/*
		if EventsArrayFunc[command] == nil { break }
		err = EventsArrayFunc[command]()
		if err != nil {
			log.Println(err)
			break
		}
		*/

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
	case ACTION_CONNECTION_TURN_OK:
		log.Println("case 2:", data)
		log.Println(string(data[:7]))
		for k, _ := range comingConns {
			if k.RemoteAddr() == conn.RemoteAddr() {
				return
			}
		}

		log.Println("节点连接成功")
		peers[conn] = true

	//	go handleTCPConnection(conn, event, processLogic)

		err := event.OnOK(command)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		if EventsArrayFunc[command] == nil { break }
		err := EventsArrayFunc[command]()
		if err != nil {
			log.Println(err)
			break
		}
	*/


	/*
		穿透服务收到打洞回馈信息
		连接关系:
		source:A distination:S data:B
	*/
	case ACTION_CONNECTION_TURNING:
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

		var connStB *net.TCPConn
		for k, _ := range comingConns {
			if k.RemoteAddr().String() == rAddrC.String() {
				connStB = k
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
		n, err := connStB.Write(sendData)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println("n:", n)

		err = event.OnTurning(command)
		if err != nil {
			log.Println(err)
			break
		}

		/*
		if EventsArrayFunc[command] == nil { break }
		err = EventsArrayFunc[command]()
		if err != nil {
			log.Println(err)
			break
		}
		*/

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
		sendData = append(sendData, intToBytes(ACTION_CONNECTION_TURN_OK)...)
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, body...)

	//	var tf bool
		for k, _ := range peers {
			if k.RemoteAddr().Network() == rAddrC.Network() && k.RemoteAddr().String() == rAddrC.String() {
				log.Println("B has A:", k.LocalAddr(), k.RemoteAddr())
				n, err := k.Write(sendData)
				if err != nil {
					log.Println(err)
					break
				}
				log.Println("n:", n)

			//	tf = true
				break
			}
		}

		/*
		if tf {
			lAddr, err := net.ResolveTCPAddr(conn.LocalAddr().Network(), conn.LocalAddr().String())
			if err != nil {
				log.Println(err)
			}

			d := net.Dialer {Control: controlSockReusePortUnix, LocalAddr: lAddr}
			connc, err := d.Dial(rAddrC.Network(), rAddrC.String())
			if err != nil {
				log.Println(err)
				break
			}
		}
		*/

		err := event.OnNotice2(command)
		if err != nil {
			log.Println(err)
			break
		}

		/*
		if EventsArrayFunc[command] == nil { break }
		err = EventsArrayFunc[command]()
		if err != nil {
			log.Println(err)
			break
		}
		*/

	/*
		点对点正式通信，此时穿透服已不需要
		连接关系:
		source:A distination:B
		or
		source:B distination:A
	*/
	case ACTION_CONNECTION_LOGIC:
		log.Println("case 5:", data)

		api, err := bytesToInt(data[:4])
		if err != nil {
			log.Println(err)
			break
		}

		log.Println("api:", api)
		log.Println(data[:4], "--", data[4:])
		err = processLogic(api, data[4:], conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		用于测试TCP
	*/
	case 0xffffffff:
		log.Println(string(data))
	}
}

/*
	两个 peer 连接后用此方法发送数据

	conn 是与对方的连接
	api 是被对方识别的功能号
	body 是从某个struct转来的字节切片

	如果发送时有任何网络错误将返回一个 error 类型的值
	否则返回 nil
*/
func Send(conn *net.TCPConn, api int, body []byte) error {
	data := append(intToBytes(api), body...)

	err := send(conn, data)
	if err != nil {
		delete(peers, conn)
		return err
	}

	return nil
}

/*
	Broadcast 函数相当于多次执行了Send方法
	达到向所有存在的 peer 发送相同的数据

	如果某一次个发送出现了错误，他将打印这个错误
	但是不会退出，然后继续向之后的连接发送消息
*/
func Broadcast(api int, body []byte) {
	var err error
	data := append(intToBytes(api), body...)

	for conn, _ := range peers {
		err = send(conn, data)
		if err != nil {
			delete(peers, conn)
			log.Println(err)
			continue
		}
	}
}

func send(conn *net.TCPConn, data []byte) error {
	sendData := []byte(PACKET_IDENTIFY)
	sendData = append(sendData, intToBytes(ACTION_CONNECTION_LOGIC)...)
	sendData = append(sendData, intToBytes(len(data))...)
	sendData = append(sendData, data...)
	_, err := conn.Write(sendData)
	if err != nil {
		return err
	}

	return nil
}

func handlePanic(funcname string) {
	err := recover()
	if err != nil {
		log.Println("in", funcname, "panic:", err)
	}
}
