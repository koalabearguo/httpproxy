package main

import "net"

import "net/http"
import "log"

//import "fmt"
//import "net/url"
import "bytes"

//import "strings"
import "io"

//import "time"
import "crypto/tls"

func main() {

	listen := "127.0.0.1:8081"
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Listening on " + listen)

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleClientRequest(client)
	}

}
func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}

	defer client.Close()

	var b [http.DefaultMaxHeaderBytes / 512]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	var firstline string = string(b[:bytes.IndexByte(b[:], '\r')])
	log.Println(firstline)

	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "www.aliyun.com",
		MinVersion:         tls.VersionTLS13,
	}

	server, err := tls.Dial("tcp", "s.koalabear.tk:443", conf)

	if err != nil {
		log.Println(err)
		return
	}

	defer server.Close()

	cnt, err := server.Write(b[:n])
	if err != nil {
		log.Println(cnt, err)
		return
	}

	//exchange data
	go io.Copy(server, client)
	io.Copy(client, server)

	return
}
