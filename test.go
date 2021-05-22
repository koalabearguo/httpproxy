package main

import "bufio"
import "net/http"
import "log"
import "strings"

func main() {
b :="GET /index.php HTTP/1.1\r\nHost: server.example.com:80\r\nProxy-Authorization: basic aGVsbG86d29ybGQ=\r\n\r\n"

reader := bufio.NewReader(strings.NewReader(b))
	Req, err := http.ReadRequest(reader)
	if err != nil {
		log.Fatal(err)
	}
	
	log.Println(Req.Method,Req.URL.Host,Req.Proto)
c :=[]byte(`abgagahahaha\r\n`)
log.Println(string(c[:]))
}