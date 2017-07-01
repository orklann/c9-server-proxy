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
	Status     string
	Address    string
}

func (h *HTTPServerConn) read() {
	for {
		data := make([]byte, 1024*1024*10)
		read, err := h.RemoteConn.Read(data)
		if err != nil {
			fmt.Printf("%s\n", "Remote closed connection")
			h.Status = Closed
			h.RemoteConn.Close()
			break
		}
		h.Data = append(h.Data, data[:read])
		fmt.Printf("Data len: %d\n", len(h.Data))
	}
}

var (
	connections = make(map[string]*HTTPServerConn)
)

// Connection Status
const (
	Connected = "C"
	NoData    = "N"
	Closed    = "X"
	Sent      = "S"
	Data      = "D"
)

// Connection actions
const (
	Connect = "C"
	Read    = "R"
	Send    = "S"
	Close   = "X"
)

func lookupStatus(s string) string {
	fmt.Printf("Status: %s\n", s)
	if s == Connect {
		return "Connected"
	} else if s == NoData {
		return "NoData"
	} else if s == Closed {
		return "Closed"
	} else if s == Sent {
		return "Sent"
	} else if s == Data {
		return "Data"
	}

	return "Unknown"
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

	return ver, addr, dst, l + 1
}

func connsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Current connections: %d", len(connections))
}

func handler(w http.ResponseWriter, r *http.Request) {
	//fmt.Printf("New Request to %s\n", r.RequestURI)

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		fmt.Fprintf(w, "%s", err)
		return
	}

	ver, key, dst, l := parseRequest(body)

	//fmt.Printf("Ver: %s Dst: %s len: %d key:%s\n", ver, dst, l, key)

	if ver == Connect {
		if _, ok := connections[key]; !ok {
			remote, err := net.Dial("tcp", dst)
			if err != nil {
				errMsg := "Failt to connect: " + err.Error()
				fmt.Fprintf(w, "%s", errMsg)
				fmt.Printf("%s\n", errMsg)
				return
			}
			httpServerConn := &HTTPServerConn{Address: key, RemoteConn: remote}
			connections[key] = httpServerConn
			go httpServerConn.read()
		}
		fmt.Printf("Connections: %d\n", len(connections))
		w.Write([]byte(Connected))
	} else if ver == Send {
		if httpServerConn, ok := connections[key]; ok {
			//fmt.Printf("***[%s]***\n", string(body[:]))
			httpServerConn.RemoteConn.Write(body[l:])
			w.Write([]byte("Sent"))
		}
	} else if ver == Close {
		if httpServerConn, ok := connections[key]; ok {
			httpServerConn.RemoteConn.Close()
			delete(connections, key)
			w.Write([]byte("Closed"))
		}
	} else if ver == Read {
		if httpServerConn, ok := connections[key]; ok {
			if len(httpServerConn.Data) > 0 {
				status := Data
				buf := append([]byte(status), httpServerConn.Data[0]...)
				httpServerConn.Data = httpServerConn.Data[1:]
				w.Write(buf)
				return
			} else if len(httpServerConn.Data) == 0 {
				status := NoData
				w.Write([]byte(status))
				return
			}

			if len(httpServerConn.Data) == 0 && httpServerConn.Status == Closed {
				status := Closed
				w.Write([]byte(status))
				return
			}
		}

	}

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
	http.ListenAndServe("0.0.0.0:"+port, nil)
}
