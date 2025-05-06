// OMSAY Server
package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

var clients = make(map[net.Conn]string)
var mu sync.Mutex

func main() {
	go handleDiscovery()

	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Println("OMSAY server running on port 8000")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()

	mu.Lock()
	clients[conn] = addr
	mu.Unlock()

	joinMsg := fmt.Sprintf("📡 %s joined OMSAY server\n", addr)
	broadcast(joinMsg, nil) // broadcast to all, including sender
	fmt.Print(joinMsg)

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		broadcast(msg, conn)
	}

	mu.Lock()
	delete(clients, conn)
	mu.Unlock()

	leaveMsg := fmt.Sprintf("🚪 %s left the chat\n", addr)
	broadcast(leaveMsg, nil)
	fmt.Print(leaveMsg)
}

func broadcast(msg string, sender net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	for conn := range clients {
		if conn != sender {
			conn.Write([]byte(msg))
		}
	}
}

func handleDiscovery() {
	addr, _ := net.ResolveUDPAddr("udp", ":9000")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, clientAddr, _ := conn.ReadFromUDP(buf)
		if strings.TrimSpace(string(buf[:n])) == "DISCOVER" {
			localIP := getLocalIP()
			replyAddr := &net.UDPAddr{IP: clientAddr.IP, Port: 9001}
			conn.WriteToUDP([]byte(localIP), replyAddr)
		}
	}
}

func getLocalIP() string {
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
