package main

import (
	"fmt"
	"io"
	"net"
	"time"
	
)

type tdsSession struct {
	buf          *tdsBuffer
	loginAck     loginAckStruct
	database     string
	partner      string
	tranid       uint64
	routedServer string
	routedPort   uint16
}

type Auth interface {
	InitialBytes() ([]byte, error)
	NextBytes([]byte) ([]byte, error)
	Free()
}

func main(){
//	connect()
}

//token.go

type loginAckStruct struct {
	Interface  uint8
	TDSVersion uint32
	ProgName   string
	ProgVer    uint32
}

//end token.go

//buf.go
func newTdsBuffer(bufsize int, transport io.ReadWriteCloser) *tdsBuffer {
	buf := make([]byte, bufsize)
	w := new(tdsBuffer)
	w.buf = buf
	w.pos = 8
	w.transport = transport
	w.size = 0
	return w
}

type tdsBuffer struct {
	buf         []byte
	pos         uint16
	transport   io.ReadWriteCloser
	size        uint16
	final       bool
	packet_type uint8
	afterFirst  func()
}
//end buf.go

func dialConnection() (conn net.Conn, err error){
			
	d := &net.Dialer{Timeout: 10000, KeepAlive: time.Hour}
	addr := "http://localhost:8080/tfs"
	conn, err = d.Dial("tcp", addr)
	
	if err != nil {
		f := "Unable to open tcp connection with host '%v': %v"
		return nil, fmt.Errorf(f, addr, err.Error())
	}
	

	return conn, err
}

func connect() (res *tdsSession, err error) {


initiate_connection:
	conn, err := dialConnection()
	if err != nil {
		return nil, err
	}

	toconn := NewTimeoutConn(conn, time.Hour)

	outbuf := newTdsBuffer(4096, toconn)
	sess := tdsSession{
		buf:      outbuf,
	}

	auth, auth_ok := getAuth("", "", "", "")
	if auth_ok {
		auth_sspi, err := auth.InitialBytes()
		if err != nil {
			return nil, err
		}
		defer auth.Free()
	} else {
		fmt.Println("auth_ok=false")
	}
	
	err = sendLogin(outbuf, login)
	
	if err != nil {
		return nil, err
	}

	// processing login response
	var sspi_msg []byte
continue_login:
	tokchan := make(chan tokenStruct, 5)
	go processResponse(&sess, tokchan)
	success := false
	for tok := range tokchan {
		switch token := tok.(type) {
		case sspiMsg:
			sspi_msg, err = auth.NextBytes(token)
			if err != nil {
				return nil, err
			}
		case loginAckStruct:
			success = true
			sess.loginAck = token
		case error:
			return nil, fmt.Errorf("Login error: %s", token.Error())
		}
	}
	if sspi_msg != nil {
		outbuf.BeginPacket(packSSPIMessage)
		_, err = outbuf.Write(sspi_msg)
		if err != nil {
			return nil, err
		}
		err = outbuf.FinishPacket()
		if err != nil {
			return nil, err
		}
		sspi_msg = nil
		goto continue_login
	}
	if !success {
		return nil, fmt.Errorf("Login failed")
	}
	if sess.routedServer != "" {
		toconn.Close()
		p.host = sess.routedServer
		p.port = uint64(sess.routedPort)
		goto initiate_connection
	}
	return &sess, nil
}
