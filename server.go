package main

import (
	"fmt"
	_ "io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	connections = make(map[string]net.Conn)
	//buffers     = make(map[string][]byte)
)

func releaseConnection(key string) {
	if v, ok := connections[key]; ok {
		v.Close()
		delete(connections, key)
	}
}

// Parse destination address
func parseRequest(r []byte) (key string, dst string, l int) {
	addr := ""
	for _, v := range r {
		l++
		if v == 's' || v == 'd' {
			continue
		} else if v == 'S' {
			addr += "=>"
			continue
		} else if v == 'D' {
			break
		}

		addr += string(v)
	}

	dst = strings.Split(addr, "=>")[1]

	return addr, dst, l
}

func connsHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Current connections: %d", len(connections))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("New Request to %s\n", r.RequestURI)

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}

	key, dst, l := parseRequest(body)

	fmt.Printf("Dst: %s len: %d key:%s\n", dst, l, key)

	if _, ok := connections[key]; !ok {
		remote, err := net.Dial("tcp", dst)
		if err != nil {
			errMsg := "Failt to connect: " + err.Error()
			fmt.Fprintf(w, "%s", errMsg)
			fmt.Printf("%s\n", errMsg)
			return
		}
		connections[key] = remote
	}

	fmt.Printf("Connections: %d\n", len(connections))

	remoteConn := connections[key]
	_, err = remoteConn.Write(body[l:])

	if err != nil {
		fmt.Printf("%s\n", "Write to remote fail")
		fmt.Fprintf(w, "%s", "Write to remote fail")
		return
	}

	buf := make([]byte, 0, 1024*1024*10)
	for {
		data := make([]byte, 1024*1024)
		read, err := remoteConn.Read(data)
		if err != nil {
			fmt.Printf("%s\n", "Remote closed connection")
			//fmt.Fprintf(w, "%s", "CLOSED")
			//releaseConnection(key)
			break
		}
		fmt.Printf("Read %d\n", read)
		buf = append(buf, data[:read]...)
	}

	w.Write(buf)
	fmt.Printf("Response %d bytes\n", len(buf))
	// Doc: https://golang.org/pkg/net/http/
	// For incoming requests, the Host header is promoted to the
	// Request.Host field and removed from the Header map.
	// host := r.Host
	//fmat.Fprintf(w, "%x", rData[:totalRead])
}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/proxy", handler)
	http.HandleFunc("/conn", connsHandler)

	fmt.Printf("Listen on port: %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
