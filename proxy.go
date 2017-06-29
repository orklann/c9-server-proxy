package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"pf"
	"strconv"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.96 Safari/537.36"

var (
	testing  = true
	host     = "www.google.com"
	postURL  = "http://localhost:8080/proxy"
	testURL  = "http://localhost:8080/proxy"
	testHost = "electric-abode-166904.appspot.com"
)

type Server struct {
	Host string `json:"host"`
	URL  string `json:"url"`
}

func getServer() Server {
	raw, err := ioutil.ReadFile("./proxy.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var s Server
	json.Unmarshal(raw, &s)
	return s
}

func info(msg string) {
	fmt.Printf("INFO - %s", msg)
}

// Post binary data to url
func postBytes(method string, url string, data []byte, host string) ([]byte, error) {
	body := bytes.NewReader(data)
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Println("http.NewRequest,[err=%s][url=%s]", err, url)
		return []byte(""), err
	}

	// Doc: https://golang.org/pkg/net/http/
	// For incoming requests, the Host header is promoted to the
	// Request.Host field and removed from the Header map.
	request.Host = host
	request.Header.Add("User-Agent", userAgent)
	if method == http.MethodPost {
		request.Header.Add("Content-Type", "application/octet-stream")
	}

	var resp *http.Response

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	http.DefaultClient.Transport = tr
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, url)
		return []byte(""), err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, url)
	}
	return b, err
}

func clientToHTTP(conn net.Conn, address string) {
	addr := []byte(address)
	for {
		data := make([]byte, 1024*1024)
		read, err := conn.Read(data)
		if err != nil {
			fmt.Println("Client closed connection")
			conn.Close()
		}

		fmt.Printf("***\n%s\n**END**\n", string(data[:read]))
		buf := append(addr, data[:read]...)
		// send data via HTTP Post
		rData, err := postBytes(http.MethodPost, postURL, buf, host)

		if err != nil {
			fmt.Println("Error response from HTTP Server")
		}

		fmt.Printf("Get %d bytes from HTTP Server\n", len(rData))
		conn.Write(rData)

	}
}

func handleConn(conn net.Conn, src net.IP, srcPort int, dst net.IP, dstPort int) {
	address := "s" + src.String() + ":" + strconv.Itoa(srcPort) + "S"
	address += "d" + dst.String() + ":" + strconv.Itoa(dstPort) + "D"

	fmt.Printf("Address len: %d\n", len(address))
	go clientToHTTP(conn, address)
}

func main() {
	info("Starting server...\n")

	server := getServer()
	host = server.Host
	postURL = server.URL

	if testing {
		host = testHost
		postURL = testURL
	}

	fmt.Printf("URL: %s\n", postURL)
	fmt.Printf("Host: %s\n", host)

	// Listen on 127.0.0.1:11235
	ln, _ := net.Listen("tcp", "0.0.0.0:11235")

	for {

		conn, _ := ln.Accept()

		srcAddr := conn.RemoteAddr()
		destAddr := conn.LocalAddr()

		srcIP := srcAddr.(*net.TCPAddr).IP
		srcPort := srcAddr.(*net.TCPAddr).Port

		destIP := destAddr.(*net.TCPAddr).IP
		destPort := destAddr.(*net.TCPAddr).Port

		rIP, rPort, err := pf.QueryNat(pf.AF_INET, pf.IPPROTO_TCP, srcIP, srcPort, destIP, destPort)

		if err != nil {
			fmt.Println("Query Nat fail!")
			continue
		}

		fmt.Println("Handle connection:" + conn.RemoteAddr().String() + "=>" + rIP.String() + ":" + strconv.Itoa(rPort))
		go handleConn(conn, srcIP, srcPort, rIP, rPort)
	}
}
