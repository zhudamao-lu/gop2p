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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"context"
	"runtime"
	"sync"
	"net"
	"time"
	"fmt"
	"log"
)

const (
	ACTION_CONNECTION_REQUEST = uint8(0x00) // 接入请求
	ACTION_CONNECTION_RESPONSE = uint8(0x01) // 接入返回
	ACTION_CONNECTION_NOTICE = uint8(0x02) // 新增接入通知，并开始穿透
	ACTION_CONNECTION_TURN_OK = uint8(0x03) // 收到则穿透成功
	ACTION_CONNECTION_TURNING = uint8(0x04) // 服务器被告知已做过一次穿透，并通知另一客户端反向访问
	ACTION_CONNECTION_NOTICE2 = uint8(0x05) // 第二次穿透访问
	ACTION_CONNECTION_LOGIC = uint8(0x06) // peer to peer 之间处理正式通信内容

	PACKET_IDENTIFY = "--TCPHEADX--" // 包开始标识符
	PACKET_IDENTIFY_LEN = 0xc // 包开始标识符长度 12
	PACKET_COMMAND_LEN = 0x1 // 包头中命令固定长度 1
	PACKET_COMMAND_END_LEN = 0xd // 包头中开始到命令结束固定长度 12 + 1
	PACKET_BODY_SIZE_LEN = 0x8 // 包头中记录数据的固定长度 8
	PACKET_BODY_SIZE_END_LEN = 0x15 // 包头中从开始到记录数据的固定长度结束的长度 12 + 1 + 8
	PACKET_TIME_LEN = 0x8 // 时间长度 8
	PACKET_TIME_END_LEN = 0x1d // 包头中开始到时间结束长度 12 + 1 + 8 + 8
	PACKET_NONCE_LEN= 0x4 // 随机数长度 4
	PACKET_NONCE_END_LEN= 0x21 // 包头中开始到随机数结束长度 12 + 1 + 8 + 8 + 4
	PACKET_HASH_LEN= 0x20 // 包头hash长度
	PACKET_HEAD_LEN = 0x41 // 包头固定长度 12 + 1 + 8 + 8 + 4 + 32
)

type hashNonce_T struct {
	hash []byte
	nonce []byte
	timestamp int64
	prev *hashNonce_T
	next *hashNonce_T
}

var hashNonceRWMutex = &sync.RWMutex{}

func (hn *hashNonce_T) countDown() {
	time.Sleep(time.Second * 300)
	hashNonceRWMutex.Lock()
	hn.prev = hn.next
	if hn == hashNonceFirst {
		hashNonceFirst = hashNonceFirst.next
	}
	hashNonceRWMutex.Unlock()
	hn = nil
}

type ConnStatus_T struct{
	B bool
	C chan int
}

var (
	seedAddrs = make(map[*net.TCPAddr]bool, 4) // 种子节点地址map
	comingConns = make(map[*net.TCPConn]bool, 12) // 自身看作穿透服时，新连接map
	peers = make(map[*net.TCPConn]*ConnStatus_T, 16) // 自身看作客户端时，新连接map
	hashNonceFirst *hashNonce_T
	hashNonceCurrent *hashNonce_T
	MaxConnection = 16
	justSignalServer = false
	totalSecondCount int
	ConnsSeedsRWMutex = &sync.RWMutex{}
)

type Event_T struct {
	Args [6]interface{}
	OnRequest func(command uint8, innerArgs ...interface{}) error
	OnResponse func(command uint8, innerArgs ...interface{}) error
	OnNotice func(command uint8, innerArgs ...interface{}) error
	OnOK func(command uint8, innerArgs ...interface{}) error
	OnTurning func(command uint8, innerArgs ...interface{}) error
	OnNotice2 func(command uint8, innerArgs ...interface{}) error
	OnDisconnect func(peer *net.TCPConn)
}

/*
	Event 默认事件全局变量
*/
var Event *Event_T

/*
	ProcessLogic 是在穿透成功建立点对点连接
*/
var ProcessLogic func([]byte, []byte, *net.TCPConn)

func init() {}

func GetPeers() map[*net.TCPConn]*ConnStatus_T {
	return peers
}

func AddPeer(conn *net.TCPConn) {
	connStatus := ConnStatus_T{
		C: make(chan int),
		B: true,
	}
	peers[conn] = &connStatus
}

func RemovePeer(conn *net.TCPConn) {
	defer ConnsSeedsRWMutex.Unlock()
	ConnsSeedsRWMutex.Lock()
	conn.Close()
	for k, _ := range seedAddrs {
		if hex.EncodeToString(k.IP) == hex.EncodeToString(conn.RemoteAddr().(*net.TCPAddr).IP) && k.Port == conn.RemoteAddr().(*net.TCPAddr).Port && k.Zone == conn.RemoteAddr().(*net.TCPAddr).Zone {
			delete(seedAddrs, k)
		}
	}
	delete(comingConns, conn)
	delete(peers, conn)
	conn = nil
}

func GetSeedAddrs() map[*net.TCPAddr]bool {
	return seedAddrs
}

/*
	连接种子节点
*/
func connectSeed(lAddr *net.TCPAddr, seedAddrsStr []string, isSync bool) error {
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
			delete(seedAddrs, k)
			fmt.Println(len(seedAddrs))
			continue
		}
		log.Println(connc.LocalAddr())

		body := []byte("hello server")
		data := []byte(PACKET_IDENTIFY)
		data = append(data, byte(ACTION_CONNECTION_REQUEST))
		data = append(data, intToBytes(len(body))...)
		data = append(data, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		data = append(data, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		data = append(data, make([]byte, 32, 32)...) // hash
		data = append(data, body...)

		go handleTCPConnection(connc.(*net.TCPConn), isSync)
		_, err = connc.Write(data)
		if err != nil {
			Event.OnDisconnect(connc.(*net.TCPConn))
			log.Println(err)
			continue
		}
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

	并且收到对方节点传来的数据时，根据起第一个参数(int 类型，可看作API号),
	做出的相应动作. 第一个参数将会是对方传来数据的body中前4个字节.
	该func 由用户实现, 并传入 StartTCPTurnServer.
*/
func StartTCPTurnServer(seedAddrsStr []string, isSync bool) error {
	var listenConfig net.ListenConfig
	listenConfig = net.ListenConfig{Control: controlSockReusePortUnix}

	ln, err := listenConfig.Listen(context.Background(), "tcp", ":11111")
	if err != nil {
		return err
	}
	fmt.Println(ln.Addr())

	lAddr, err := net.ResolveTCPAddr(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		return err
	}
	log.Println(lAddr)

	connectSeed(lAddr, seedAddrsStr, isSync)

	listenAccept(ln, isSync)

	return nil
}

func handleTCPConnection(conn *net.TCPConn, isSync bool) {
	defer conn.Close()

	data := make([]byte, 0, 4096)

	defer func() {
		err := recover()
		if err != nil {
			log.Println(err)
			buf := make([]byte, 2048)

			n := runtime.Stack(buf, false)
			fmt.Println(string(buf[:n]))
		}
	}()
	var command uint8
	var headForHash []byte
	var bodyLength int
	var hashNonce *hashNonce_T

	go func(){
		for {
			time.Sleep(1 * time.Second)
			if totalSecondCount == 0 {
				continue
			}
			if peers[conn] != nil {
				continue
			}
			_, ok := peers[conn]
			if ok == false {
				continue
			}
			peers[conn].C <- totalSecondCount
			totalSecondCount = 0
		}
	}()

	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			RemovePeer(conn)
			Event.OnDisconnect(conn)
			log.Println(err)
			break
		}
		totalSecondCount += n

		data = append(data, buffer[:n]...)

		if len(data) < PACKET_IDENTIFY_LEN {
			continue
		}

		for len(data) >= PACKET_HEAD_LEN && string(data[:PACKET_IDENTIFY_LEN]) == PACKET_IDENTIFY {
			command, headForHash, bodyLength, hashNonce, err = decodeData(data)
			if err != nil {
				log.Println(err)
				data = data[:]
				break
			}

			bodyEnd := PACKET_HEAD_LEN + bodyLength
			if bodyEnd <= len(data) {
				body := make([]byte, bodyLength, bodyLength)
				copy(body, data[PACKET_HEAD_LEN : bodyEnd])
				tcpHandle(command, headForHash, body, hashNonce, conn, isSync)
				data = data[bodyEnd :]
				continue
			}

			break
		}
	}
}

func listenAccept(ln net.Listener, isSync bool) {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(conn.RemoteAddr())

		go handleTCPConnection(conn.(*net.TCPConn), isSync)
	}
}

func decodeData(data []byte) (uint8, []byte, int, *hashNonce_T, error) {
	command := uint8(data[PACKET_IDENTIFY_LEN])

	bodyLength, err := bytesToInt(data[PACKET_COMMAND_END_LEN : PACKET_BODY_SIZE_END_LEN])
	if err != nil {
		data = data[0:0]
		return 0, nil, 0, nil, err
	}

	timestamp, err := bytesToInt64(data[PACKET_BODY_SIZE_END_LEN : PACKET_TIME_END_LEN])
	if err != nil {
		data = data[0:0]
		return 0, nil, 0, nil, err
	}

	hashNonce := &hashNonce_T{make([]byte, PACKET_HASH_LEN, PACKET_HASH_LEN), make([]byte, PACKET_NONCE_LEN, PACKET_NONCE_LEN), timestamp, nil, nil}

	copy(hashNonce.nonce, data[PACKET_TIME_END_LEN : PACKET_NONCE_END_LEN])
	copy(hashNonce.hash, data[PACKET_NONCE_END_LEN : PACKET_HEAD_LEN])

	if command == ACTION_CONNECTION_LOGIC {
		headForHash := make([]byte, PACKET_NONCE_END_LEN - PACKET_IDENTIFY_LEN, PACKET_NONCE_END_LEN - PACKET_IDENTIFY_LEN)
		copy(headForHash, data[PACKET_IDENTIFY_LEN : PACKET_NONCE_END_LEN])

		return command, headForHash, bodyLength, hashNonce, nil
	}

	return command, nil, bodyLength, hashNonce, nil
}

func tcpHandle(command uint8, headForHash, data []byte, hashNonce *hashNonce_T, conn *net.TCPConn, isSync bool) {
//	defer handlePanic("tcpHandle")

	switch command {
	/*
		穿透服收到接入请求
		连接关系:
		source:B distination:S
	*/
	case ACTION_CONNECTION_REQUEST:
		fmt.Println("case 0:")
		if !justSignalServer {
			if len(comingConns) >= MaxConnection || len(peers) >= MaxConnection {
				fmt.Println("connection number out of bounds")
				break
			}
		}

		/*
		for k, v := range comingConns {
			fmt.Println(k.LocalAddr(), k.RemoteAddr(), v)
		}
		*/

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
			sendData = append(sendData, byte(ACTION_CONNECTION_NOTICE))
			sendData = append(sendData, intToBytes(len(body))...)
			sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
			sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
			sendData = append(sendData, make([]byte, 32, 32)...) // hash
			sendData = append(sendData, body...)
			_, err := k.Write(sendData)
			if err != nil {
				Event.OnDisconnect(k)
				log.Println(err)
				continue
			}
		}

		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, byte(ACTION_CONNECTION_RESPONSE))
		sendData = append(sendData, []byte{0, 0, 0, 0, 0, 0, 0, 0}...)
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		sendData = append(sendData, make([]byte, 32, 32)...) // hash
		_, err = conn.Write(sendData)
		if err != nil {
			Event.OnDisconnect(conn)
			log.Println(err)
		}

		comingConns[conn] = true
		fmt.Println(string(data))

		err = Event.OnRequest(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		自己作为新节点收到穿透服发来的确认加入信息
		连接关系:
		source:S distination:B
	*/
	case ACTION_CONNECTION_RESPONSE:
		fmt.Println("case 1:")
		err := Event.OnResponse(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		收到穿透服务器发来的，新节点加入信息
		对他进行访问用来打洞，并告知穿透服这一操作
		连接关系:
		source:S distination:A data:B
	*/
	case ACTION_CONNECTION_NOTICE:
		fmt.Println("case 2:")

		if len(peers) >= MaxConnection {
			return
		}

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.TCPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}

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

		if connc == nil {
			log.Println("connc is nil")
			return
		}

		go handleTCPConnection(connc.(*net.TCPConn), isSync)

		body := []byte("turn...")
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, byte(ACTION_CONNECTION_TURN_OK))
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		sendData = append(sendData, make([]byte, 32, 32)...) // hash
		sendData = append(sendData, body...)
		_, err = connc.Write(sendData)
		if err != nil {
			Event.OnDisconnect(connc.(*net.TCPConn))
			log.Println(err)
			break
		}

		// 告知穿透服已打洞
		body = []byte("turned...")
		body = append(body, data...)
		sendData = []byte(PACKET_IDENTIFY)
		sendData = append(sendData, byte(ACTION_CONNECTION_TURNING))
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		sendData = append(sendData, make([]byte, 32, 32)...) // hash
		sendData = append(sendData, body...)
		_, err = conn.Write(sendData)
		if err != nil {
			Event.OnDisconnect(conn)
			log.Println(err)
			break
		}

		err = Event.OnNotice(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

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
		fmt.Println("case 3:")
		fmt.Println(string(data[:7]))
		for k, _ := range comingConns {
			if k.RemoteAddr() == conn.RemoteAddr() {
				return
			}
		}

		fmt.Println("节点连接成功")
		_, ok := peers[conn]
		if ok == true {
			peers[conn].B = true
		} else {
			log.Println("peers this conn has been delete")
		}

		err := Event.OnOK(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		穿透服务收到打洞回馈信息
		连接关系:
		source:A distination:S data:B
	*/
	case ACTION_CONNECTION_TURNING:
		fmt.Println("case 4:")
		fmt.Println(string(data[:9]))

		// Decode 对方客户端 addr
		ip := net.IP(data[9:25])
		rAddrC := &net.TCPAddr{ip, int(data[25]) << 8 | int(data[26]), ""}
		fmt.Println(rAddrC)

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
		sendData = append(sendData, byte(ACTION_CONNECTION_NOTICE2))
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		sendData = append(sendData, make([]byte, 32, 32)...) // hash
		sendData = append(sendData, body...)
		_, err = connStB.Write(sendData)
		if err != nil {
			Event.OnDisconnect(connStB)
			log.Println(err)
			break
		}

		err = Event.OnTurning(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		客户端收到穿透服告知其他客户端已向自己打洞
		开始向其他客户端发送消息
		连接关系:
		source:S distination:B data:A
	*/
	case ACTION_CONNECTION_NOTICE2:
		fmt.Println("case 5:")

		// Decode 对方客户端 addr
		ip := net.IP(data[:16])
		rAddrC := &net.TCPAddr{ip, int(data[16]) << 8 | int(data[17]), ""}
		fmt.Println(rAddrC)

		// 封装数据 发送 "turn..."
		body := []byte("turn...")
		sendData := []byte(PACKET_IDENTIFY)
		sendData = append(sendData, byte(ACTION_CONNECTION_TURN_OK))
		sendData = append(sendData, intToBytes(len(body))...)
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}...) // timestamp
		sendData = append(sendData, []byte{0xff, 0xff, 0xff, 0xff}...) // nonce
		sendData = append(sendData, make([]byte, 32, 32)...) // hash
		sendData = append(sendData, body...)

		for k, _ := range peers {
			if k.RemoteAddr().Network() == rAddrC.Network() && k.RemoteAddr().String() == rAddrC.String() {
				fmt.Println("B has A:", k.LocalAddr(), k.RemoteAddr())
				_, err := k.Write(sendData)
				if err != nil {
					Event.OnDisconnect(k)
					log.Println(err)
					break
				}

				break
			}
		}

		err := Event.OnNotice2(command, conn)
		if err != nil {
			log.Println(err)
			break
		}

	/*
		点对点正式通信，此时穿透服已不需要
		连接关系:
		source:A distination:B
		or
		source:B distination:A
	*/
	case ACTION_CONNECTION_LOGIC:
		fmt.Println("case 6:")

		sum := sha256.Sum256(headForHash)
		if hex.EncodeToString(sum[:]) != hex.EncodeToString(hashNonce.hash) {
			fmt.Println("invalid hash:")
			data = data[:]
			break
		}

		if hashNonce.timestamp + 300000000000 < time.Now().UnixNano() {
			fmt.Println("expired packet")
			data = data[:]
			break
		}

		hashNonceBytes := make([]byte, len(hashNonce.nonce) + len(hashNonce.hash))
		copy(hashNonceBytes, append(hashNonce.nonce, hashNonce.hash...))
		head := append([]byte("--TCPHEADX--"), append([]byte{byte(command)}, append(intToBytes(len(data)), append(int64ToBytes(hashNonce.timestamp), hashNonceBytes...)...)...)...)

		hashNonceRWMutex.Lock()
		if hashNonceCurrent != nil {
			current := hashNonceFirst
			if hex.EncodeToString(current.hash) == hex.EncodeToString(hashNonce.hash) && hex.EncodeToString(current.nonce) == hex.EncodeToString(hashNonce.nonce) {
				log.Println("duplucated message")
				data = data[:]
				hashNonceRWMutex.Unlock()
				break
			}
			current = current.next
			var b bool
			for current != hashNonceFirst {
				if hex.EncodeToString(current.hash) == hex.EncodeToString(hashNonce.hash) && hex.EncodeToString(current.nonce) == hex.EncodeToString(hashNonce.nonce) {
					b = true
					break
				}
				current = current.next
			}
			if b {
				log.Println("duplucated message")
				data = data[:]
				hashNonceRWMutex.Unlock()
				break
			}

			hashNonceCurrent.prev = hashNonceCurrent
			hashNonceCurrent.next = hashNonce
		} else {
			fmt.Println("flow else......")
			hashNonceFirst, hashNonceCurrent = hashNonce, hashNonce
		}
		hashNonce.next = hashNonceFirst
		hashNonceCurrent = hashNonce
		hashNonceRWMutex.Unlock()

		go hashNonce.countDown()

		if isSync {
			ProcessLogic(head, data, conn)
		} else {
			go ProcessLogic(head, data, conn)
		}

	/*
		用于测试TCP
	*/
	case 0xff:
		fmt.Println(string(data))
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
func Send(conn *net.TCPConn, api int32, body []byte) error {
	data := append(int32ToBytes(api), body...)

	timestamp := time.Now().UnixNano() // timestamp

	maxBytes := []byte{1, 0, 0, 0, 0}
	max := big.NewInt(0)
	max.SetBytes(maxBytes)
	random, err := rand.Int(rand.Reader, max)
	if err != nil {
		return err
	}

	err = send(conn, timestamp, random.Bytes(), data)
	if err != nil {
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
func Broadcast(api int32, body []byte) {
	var err error
	data := append(int32ToBytes(api), body...)

	timestamp := time.Now().UnixNano() // timestamp

	maxBytes := []byte{1, 0, 0, 0, 0}
	max := big.NewInt(0)
	max.SetBytes(maxBytes)
	random, err := rand.Int(rand.Reader, max)
	if err != nil {
		log.Println(err)
		return
	}

	randomI := int32(random.Int64())
	for conn, _ := range peers {
		err = send(conn, timestamp, int32ToBytes(randomI), data)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func send(conn *net.TCPConn, timestamp int64, random, data []byte) error {
	sendData := []byte(PACKET_IDENTIFY)
	sendData = append(sendData, byte(ACTION_CONNECTION_LOGIC))
	sendData = append(sendData, intToBytes(len(data))...)
	sendData = append(sendData, int64ToBytes(timestamp)...) // timestamp
	sendData = append(sendData, random...) // nonce
	hash := sha256.Sum256(sendData[PACKET_IDENTIFY_LEN : PACKET_NONCE_END_LEN])
	sendData = append(sendData, hash[:]...) // hash
	sendData = append(sendData, data...)

	_, err := conn.Write(sendData)
	if err != nil {
		Event.OnDisconnect(conn)
		log.Println(err)
		return err
	}

	return nil
}

/*
	Forward the incoming packet by broadcasting.
	param data is the packet content
*/
func Forward(data []byte, except *net.TCPConn) {
	for conn, _ := range peers {
		if conn == except {
			continue
		}

		_, err := conn.Write(data)
		if err != nil {
			Event.OnDisconnect(conn)
			log.Println(err)
			continue
		}
	}
}

/*
	GetApiFromBody will return an api number and error.
	param body is the body part of the incoming packet content.
*/
func GetApiFromBody(body []byte) (int32, error) {
	api, err := bytesToInt32(body[:4])
	if err != nil {
		return 0, err
	}

	return api, nil
}

/*
	GetDataFromBody will return an body without api number and error.
	param body is the body part of the incoming packet content.
*/
func GetDataFromBody(body []byte) []byte {
	data := make([]byte, len(body) - 4, len(body) - 4)
	copy(data, body[4:])
	return data
}

/*
	GetComingConns return the coming connections map.
	When the peer thought of a signal server, its incoming peer called coming-connection.
*/
func GetComingConns() map[*net.TCPConn]bool {
	return comingConns
}

/*
func DefaultDisconnectEvent(notices []*DisconnectNotice_T) {
	for _, n := range notices {
		go func(n * DisconnectNotice_T) {
			n.NoticeCh <- n.Param
		}(n)
	}
}
*/
