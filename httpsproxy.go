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
		MaxVersion:   tls.VersionTLS13,
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

	var b [http.DefaultMaxHeaderBytes / 512]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

	var firstline string = string(b[:bytes.IndexByte(b[:], '\r')])
	log.Println(firstline)

	var method, host, version, address string
	fmt.Sscanf(firstline, "%s%s%s", &method, &host, &version)

	hostPortURL, err := url.Parse(host)
	if err != nil {
		log.Println(err)
		return
	}

	if hostPortURL.Host == "" && method != http.MethodConnect {
		log.Println("None Proxy Mode Request")
		fmt.Fprint(client, version+" 404 Not Found\r\n\r\n")
		return
	}
	if method == http.MethodConnect {
		address = hostPortURL.Scheme + ":" + hostPortURL.Opaque
	} else {
		if strings.Index(hostPortURL.Host, ":") == -1 { //host port not include,default 80
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}

	//prepare to dial
	timeout, err := time.ParseDuration("15s")
	if err != nil {
		log.Println(err)
		return
	}
	server, err := net.DialTimeout("tcp", address, timeout)

	log.Println(address)

	if err != nil {
		log.Println(err)
		return
	}
	if method == http.MethodConnect {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		server.Write(b[:n])
	}

	//exchange data
	go io.Copy(server, client)
	io.Copy(client, server)

	return
}
