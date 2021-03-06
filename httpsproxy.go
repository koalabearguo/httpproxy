package main

import "net"
import "net/http"
import "log"
import "fmt"
import "net/url"
import "bytes"
import "strings"
import "io"
import "time"
import "crypto/tls"

func main() {
	listen := "127.0.0.1:8081"

	//certPem := []byte(``)
	//keyPem := []byte(``)

	log.SetFlags(log.Lshortfile)

	cer, err := tls.LoadX509KeyPair("/etc/letsencrypt/live/koalabear.tk/fullchain.pem",
		"/etc/letsencrypt/live/koalabear.tk/privkey.pem")
	if err != nil {
		log.Println(err)
		return
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
		MinVersion:   tls.VersionTLS13,
	}

	ln, err := tls.Listen("tcp", listen, config)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Listening on " + listen)

	defer ln.Close()

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}

	defer client.Close()

	var b [http.DefaultMaxHeaderBytes * 64 / 1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	reader := bufio.NewReader(strings.NewReader(string(b[:n])))
	Req, err := http.ReadRequest(reader)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(Req.Method, Req.URL, Req.Proto)

	if !Req.URL.IsAbs() && Req.URL.Host == "" && Req.Method != http.MethodConnect {
		log.Println("None Proxy Mode Request")
		fmt.Fprint(client, Req.Proto+" 404 Not Found\r\n\r\n")
		return
	}

	//prepare to dial
	timeout, err := time.ParseDuration("15s")
	if err != nil {
		log.Println(err)
		return
	}
	var address string
	if strings.Index(Req.URL.Host, ":") == -1 { //host port not include,default 80
		address = Req.URL.Host + ":http"
	} else {
		address = Req.URL.Host
	}

	log.Println(address)
	server, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		log.Println(err)
		return
	}
	if Req.Method == http.MethodConnect {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		server.Write(b[:n])
	}

	//exchange data
	go io.Copy(server, client)
	io.Copy(client, server)

	return
}
