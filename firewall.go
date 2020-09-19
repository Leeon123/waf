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
	waf_port         = "0.0.0.0:80"     //your waf address
	real_port        = "localhost:1337" //your real address
	rps_per_ip_limit = 100              //requests per second per ip limitation
	banned_list      []string
	static_list      = map[string]int{}
	static_lock      = sync.RWMutex{}
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
		go handle(conn)
	}
}

func unban() {
	for {
		time.Sleep(time.Second * 30) //clear banned list every 30 second, might be will change the logic later
		banned_list = nil
	}
}

func monitor() {
	for {
		static_lock.Lock()
		for ip, times := range static_list {
			if times >= rps_per_ip_limit { //limit the rps
				banned_list = append(banned_list, ip)
			}
			delete(static_list, ip) //clear it every second
		}
		static_lock.Unlock()
		time.Sleep(time.Second)
	}
}

func handle(src net.Conn) {
	remoteIP := strings.Split(src.RemoteAddr().String(), ":")[0] //Get the Ip
	defer src.Close()
	if src, ok := src.(*net.TCPConn); ok {
		src.SetNoDelay(false)
	}
	var dst net.Conn
	requestsPerConnection := 0
	for {
		src.SetDeadline(time.Now().Add(10 * time.Second))
		for _, v := range banned_list {
			if v == remoteIP {
				return
			}
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
		request := buf[:n]
		if dst == nil {
			//fmt.Println("Started a connection to real server")
			dst, err = net.DialTimeout("tcp", real_port, time.Second*10)
			if err != nil {
				src.Write([]byte("HTTP/1.1 503 service unavailable\r\n\r\n"))
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
		src.SetDeadline(time.Now().Add(10 * time.Second))
		//fmt.Println(request)// we can filter request later, such as some injection or exploit...
		requestsPerConnection++
		static_lock.Lock()
		static_list[remoteIP]++
		static_lock.Unlock()
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
