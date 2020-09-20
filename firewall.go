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
	"strings"
	"sync"
	"time"
	"unsafe"
)

var (
	// You can edit this
	waf_port         = "0.0.0.0:80"     //your waf address
	real_port        = "localhost:1337" //your real address
	pps_per_ip_limit = 50               //packets per second per ip limitation
	connection_limit = 10               //Limit the connections of one ip
	banned_time      = 60

	//You better know what are this
	connection_per_ip sync.Map //changed to sync.Map because map is unsafe
	rps_per_ip        sync.Map //changed to sync.Map because map is unsafe
	banned_list       sync.Map //changed to sync.Map for new method

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
		connections, ok := connection_per_ip.Load(remoteIP)
		if ok {
			if connections.(int) >= connection_limit {
				conn.Close()
			}
			connection_per_ip.Store(remoteIP, connections.(int)+1)
		} else {
			connection_per_ip.Store(remoteIP, 1)
		}

		go handle(conn, remoteIP)
	}
}

func unban() {
	for {
		banned_list.Range(func(ip, time_banned interface{}) bool {
			if used := time_banned.(time.Time); used.Second() > banned_time {
				banned_list.Delete(ip.(string))
			}
			return true
		})
		time.Sleep(time.Second) //check every second
	}
}

func monitor() {
	for {
		rps_per_ip.Range(func(ip, times interface{}) bool {
			if times.(int) >= pps_per_ip_limit { //limit the pps
				banned_list.Store(ip.(string), time.Now())
			}
			rps_per_ip.Delete(ip.(string))
			return true
		})
		time.Sleep(time.Second)
	}
}

func handle(src net.Conn, remoteIP string) {
	defer src.Close()
	defer func() {
		connections, ok := connection_per_ip.Load(remoteIP)
		if ok && connections.(int) > 0 {
			connection_per_ip.Store(remoteIP, connections.(int)-1)
		}
	}()
	if src, ok := src.(*net.TCPConn); ok {
		src.SetNoDelay(false)
	}
	var dst net.Conn
	requestsPerConnection := 0
	banned := false
	for {

		src.SetDeadline(time.Now().Add(10 * time.Second))
		banned_list.Range(func(ip, _ interface{}) bool {
			if ip == remoteIP {
				banned = true
				return false
			}
			return true
		})
		if banned {
			return
		}
		if requestsPerConnection >= 100 {
			return
		}
		buf := make([]byte, 2048) //i think you won't send a request over 2048 bytes right....?
		n, err := src.Read(buf)
		if err != nil {
			if dst != nil {
				dst.Close()
			}
			return
		}
		request := buf[:n] //TODO, check the request.
		if dst == nil {
			//fmt.Println("Started a connection to real server")
			dst, err = net.DialTimeout("tcp", real_port, time.Second*10)
			if err != nil {
				//src.Write([]byte("HTTP/1.1 503 service unavailable\r\n\r\n"))
				return
			}
			if dst, ok := dst.(*net.TCPConn); ok {
				dst.SetNoDelay(false)
			}
			go func() {
				defer dst.Close()
				io.Copy(src, dst)
			}()
		} else {
			//fmt.Println("Re use the connection")
		}
		dst.SetDeadline(time.Now().Add(10 * time.Second))
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
