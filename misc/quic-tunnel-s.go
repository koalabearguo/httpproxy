package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"github.com/armon/go-socks5"
	"github.com/quic-go/quic-go"
	"io"
	"log"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"sync"
	"time"
)

type Conn struct {
	conn    quic.Connection
	qstream quic.Stream
}

func (c *Conn) Read(b []byte) (n int, err error) {
	return c.qstream.Read(b)
}
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.qstream.Write(b)
}
func (c *Conn) Close() error {
	return c.qstream.Close()
}
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
func (c *Conn) SetDeadline(t time.Time) error {
	return c.qstream.SetDeadline(t)
}
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.qstream.SetReadDeadline(t)
}
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.qstream.SetWriteDeadline(t)
}

var bufpool sync.Pool

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	bufpool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 64*1024)
		},
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
		Allow0RTT:                      func(net.Addr) bool { return true },
	}
	listener, err := quic.ListenAddr("0.0.0.0:5900", generateTLSConfig(), qconf)
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}
		go handleAccept(conn)
	}
}

func monitor(conn quic.Connection) {
	rand_int := time.Duration(mrand.Intn(60))
	myT := time.NewTimer((120 + rand_int) * time.Second)
	<-myT.C
	conn.CloseWithError(0, "Refresh ISP UDP QoS.")
	log.Println("Connection Close for Refresh ISP UDP QoS.")
	return
}

func handleAccept(conn quic.Connection) {

	//go monitor(conn)

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Println(err)
			//conn.CloseWithError(0, err.Error())
			return
		}

		log.Println("Accepted a new QUIC Stream.")

		stream2conn := &Conn{conn, stream}
		//go handleConn(stream2conn)
		go handleConn1(stream2conn)

	}

}

func handleConn(conn net.Conn) {

	defer conn.Close()

	// Create a SOCKS5 server
	conf := &socks5.Config{Logger: nil}
	server, err := socks5.New(conf)
	if err != nil {
		log.Println(err)
		return
	}

	if err := server.ServeConn(conn); err != nil {
		//log.Println(err)
		return
	}
}

func handleConn1(conn net.Conn) {

	defer conn.Close()

	stream, err := net.Dial("tcp", "127.0.0.1:3128")
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("Open a new TCP Stream.")
	}
	//exchange data
	go Copy(stream, conn)
	Copy(conn, stream)
	log.Println("a TCP Stream Closed.")
	stream.Close()
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"h3"},
	}
}

func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufpool.Get().([]byte)
	written, err = io.CopyBuffer(dst, src, buf)
	bufpool.Put(buf)
	return written, err
}
