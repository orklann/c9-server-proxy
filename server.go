package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
)

type HTTPServerConn struct {
	RemoteConn net.Conn
	Data       [][]byte
}

var (
	connections = make(map[string]HTTPServerConn)
)

// Connection Status
const (
	Connected = "C"
	NoData    = "N"
	Closed    = "X"
	Sent      = "S"
)

// Connection actions
const (
	Connect = "C"
	Read    = "R"
	Send    = "S"
	Close   = "X"
)

func lookupStatus(s string) string {
	if s == Connect {
		return "Connected"
	} else if s == NoData {
		return "NoData"
	} else if s == Closed {
		return "Closed"
	} else if s == Sent {
		return "Sent"
	}

	return "Unknown"
}

func releaseConnection(key string) {
	if v, ok := connections[key]; ok {
		v.RemoteConn.Close()
		delete(connections, key)
	}
}

// Parse destination address
func parseRequest(r []byte) (ver string, key string, dst string, l int) {
	addr := ""
	ver = string(r[:1])
	for _, v := range r[1:] {
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

	return ver, addr, dst, l
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

	ver, key, dst, l := parseRequest(body)

	fmt.Printf("Ver: %s Dst: %s len: %d key:%s\n", ver, dst, l, key)

	if ver == Connect {
		if _, ok := connections[key]; !ok {
			remote, err := net.Dial("tcp", dst)
			if err != nil {
				errMsg := "Failt to connect: " + err.Error()
				fmt.Fprintf(w, "%s", errMsg)
				fmt.Printf("%s\n", errMsg)
				return
			}
			httpServerConn := HTTPServerConn{RemoteConn: remote, Data: make([][]byte, 100)}
			connections[key] = httpServerConn
		}
		fmt.Printf("Connections: %d\n", len(connections))

		w.Write([]byte(Connected))
	} else if ver == Send {
		if _, ok := connections[key]; ok {
			fmt.Printf("***[%s]***\n", string(body[:]))
			w.Write([]byte("Sent"))
		}
	}

	return
	/*
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
	   fmt.Printf("Response %d bytes\n", len(buf))*/
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
