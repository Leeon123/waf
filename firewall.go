/*
Coded by Leeon123
Date: 2020/9/20 18:01
It is not only for http server,
also for other tcp server.
*/
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var (
	// You can edit this
	waf_port                 = "0.0.0.0:80"     //your waf address
	real_port                = "localhost:1337" //your real address
	pps_per_ip_limit         = 10               //Limit the packets per second of ip
	connection_limit         = 10               //Limit the connections of ip
	banned_time      float64 = 60               //Blocking time of the banned ip
	//You better know what are this
	connection_per_ip sync.Map //changed to sync.Map because map is unsafe
	rps_per_ip        sync.Map //changed to sync.Map because map is unsafe
	banned_list       sync.Map //changed to sync.Map for new method

	connMap sync.Map //for counting connection
	errMsg  = "HTTP/1.1 503 service unavailable\r\nContent-Length:0\r\n\r\n"
)

func main() {

	listener, err := net.Listen("tcp", waf_port)
	if err != nil {
		panic("connection error:" + err.Error())
	}
	go unban()
	go monitor()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept Error:", err)
			continue
		}
		remoteIP := strings.Split(conn.RemoteAddr().String(), ":")[0] //Get the Ip
		if isBanned(remoteIP) {
			conn.Close()
			continue
		}
		connections, ok := connection_per_ip.Load(remoteIP)
		if ok {
			if connections.(int) >= connection_limit {
				banned_list.Store(remoteIP, time.Now())
				conn.Close()
				continue
			}
			connection_per_ip.Store(remoteIP, connections.(int)+1)
		} else {
			connection_per_ip.Store(remoteIP, 1)
		}
		connMap.Store(conn.RemoteAddr().String(), conn) //might be used in soon...
		go handle(conn, remoteIP)
	}
}

func unban() {
	for {
		banned_list.Range(func(ip, time_banned interface{}) bool {
			tmp := time_banned.(time.Time)
			if used := time.Since(tmp); used.Seconds() >= banned_time {
				banned_list.Delete(ip.(string))
			}
			return true
		})
		time.Sleep(time.Second * 1) //check every 5 second
	}
}

func monitor() {
	for {
		rps := 0
		currentConn := 0
		bannedIP := 0
		rps_per_ip.Range(func(ip, times interface{}) bool {
			rps++
			if times.(int) >= pps_per_ip_limit { //limit the pps
				banned_list.Store(ip.(string), time.Now())
			}
			rps_per_ip.Delete(ip.(string))
			return true
		})
		connMap.Range(func(addr, conn interface{}) bool {
			currentConn++
			return true
		})
		banned_list.Range(func(ip, time_banned interface{}) bool {
			bannedIP++
			return true
		})
		fmt.Printf("\rConnections: %d || Banned IP: %d || Rps: %d  ", currentConn, bannedIP, rps)
		os.Stdout.Sync()
		time.Sleep(time.Second)
	}
}

func isBanned(remoteIP string) bool {
	banned := false
	banned_list.Range(func(ip, _ interface{}) bool {
		if ip == remoteIP {
			banned = true
			return false
		}
		return true
	})
	return banned
}

func handle(src net.Conn, remoteIP string) {
	defer src.Close()
	defer func() {
		connections, ok := connection_per_ip.Load(remoteIP)
		if ok && connections.(int) > 0 {
			connection_per_ip.Store(remoteIP, connections.(int)-1)
		} else {
			connection_per_ip.Delete(remoteIP) //delete it if it equals to 0.
		}

		connMap.Delete(src.RemoteAddr().String())
	}()
	if src, ok := src.(*net.TCPConn); ok {
		src.SetNoDelay(false)
	}
	var dst net.Conn
	requestsPerConnection := 0
	for {
		src.SetDeadline(time.Now().Add(10 * time.Second)) //10 second timeout

		if isBanned(remoteIP) {
			return
		}
		if requestsPerConnection >= 50 {
			return
		}
		buf := make([]byte, 8192) //i think you won't send a packet over 65535 bits right....?
		n, err := src.Read(buf)
		if err != nil {
			if dst != nil {
				dst.Close()
			}
			return //if there are errors, just close it.
		}
		request := buf[:n] //TODO, check the request.
		if dst == nil {
			//fmt.Println("Started a connection to real server")
			dst, err = net.DialTimeout("tcp", real_port, time.Second*10)
			if err != nil {
				src.Write(str2bytes(errMsg))
				return
			}
			if dst, ok := dst.(*net.TCPConn); ok {
				dst.SetNoDelay(false)
			}
			go func() {
				defer dst.Close()
				io.Copy(src, dst) //directly transfer the data from real server to client.
			}()
		} else {
			//fmt.Println("Re use the connection")
		}
		dst.SetDeadline(time.Now().Add(10 * time.Second)) //10 second timeout
		dst.Write(request)
		//fmt.Println(request)// we can filter request later, such as some injection or exploit...
		requestsPerConnection++
		rps, ok := rps_per_ip.Load(remoteIP)
		if ok {
			rps_per_ip.Store(remoteIP, rps.(int)+1)
		} else {
			rps_per_ip.Store(remoteIP, 1)
		}
	}
}

func str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
func bytes2str(s []byte) string {
	return *(*string)(unsafe.Pointer(&s))
}
