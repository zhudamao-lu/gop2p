func genSockReusePort(network, address string) (conn *net.UDPConn, err error) {
	cfgFn := func(network, address string, conn syscall.RawConn) (err error) {
		fn := func(fd uintptr) {
			err = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, 0xf, 1) // 0xf = SO_REUSEPORT
			if err != nil {
				return
			}
		}

		err = conn.Control(fn)
		if err != nil {
			return err
		}

		return nil
	}

	lc := net.ListenConfig{Control: cfgFn}
	lp, err := lc.ListenPacket(context.Background(), "udp", address)
	if err != nil {
		return nil, err
	}
	conn = lp.(*net.UDPConn)
	return conn, nil
}
