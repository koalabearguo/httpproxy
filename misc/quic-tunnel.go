package main

import (
	"context"
	"crypto/tls"
	"github.com/quic-go/quic-go"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const num_quic_connection int = 5

var mux sync.Mutex

var bufpool sync.Pool

var ch_conn chan net.Conn

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	bufpool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024)
		},
	}

	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Println(err)
	}

	log.Println("TCP Server is running on port 8081")

	defer listener.Close()

	ch_conn = make(chan net.Conn, 10)

	for i := 0; i < num_quic_connection; i++ {
		go handleClient()
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		ch_conn <- conn
		log.Println("Accepted a new TCP connection.")

	}
}

func handleClient() {
	ch_qerr := make(chan bool)
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
	}
	qconf := &quic.Config{
		MaxIdleTimeout:                 00 * time.Second,
		HandshakeIdleTimeout:           10 * time.Second,
		KeepAlivePeriod:                0 * time.Second,
		InitialStreamReceiveWindow:     512 * 1024 * 10,
		MaxStreamReceiveWindow:         512 * 1024 * 10 * 3,
		InitialConnectionReceiveWindow: 512 * 1024 * 10,
		MaxConnectionReceiveWindow:     512 * 1024 * 10 * 9,
		EnableDatagrams:                true,
	}
	//
	for {
		qconn, err := quic.DialAddrEarly("s.koalabear.tk:443", tlsConf, qconf)
		if err != nil {
			log.Println(err)
			myT := time.NewTimer(5 * time.Second)
			<-myT.C
		} else {
			log.Println("QUIC Client Connected.")
			go handleStream(qconn, ch_qerr)
			<-ch_qerr
		}
	}
}

func handleStream(qconn quic.Connection, ch_qerr chan bool) {

	for {
		conn := <-ch_conn
		stream, err := qconn.OpenStreamSync(context.Background())
		if err != nil {
			log.Println(err)
			ch_qerr <- true
			ch_conn <- conn
			return
		} else {
			log.Println("Open a new QUIC Stream.")
		}

		go handleData(conn, stream)
	}
}

func handleData(conn net.Conn, stream quic.Stream) {
	//exchange data
	go Copy(stream, conn)
	Copy(conn, stream)
	log.Println("a QUIC Stream Closed.")
	stream.Close()
	conn.Close()
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufpool.Get().([]byte)
	written, err = io.CopyBuffer(dst, src, buf)
	bufpool.Put(buf)
	return written, err
}
