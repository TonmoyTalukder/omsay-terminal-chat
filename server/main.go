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

func genUsername(addr string) string {
	hash := 0
	for _, c := range addr {
		hash = int(c) + ((hash << 5) - hash)
	}

	prefixes := []string{"Ξ", "λ", "Ω", "ψ", "Σ", "∇", "π", "δ"}
	emojis := []string{"🧬", "⚡", "🛸", "🧠", "🚀", "👾", "💻", "🌐"}

	emoji := emojis[hash%len(emojis)]
	prefix := prefixes[(hash/len(emojis))%len(prefixes)]

	// Extract a funky hex-based suffix
	hex := fmt.Sprintf("%08X", hash)
	codename := hex[len(hex)-4:]

	return fmt.Sprintf("%s%s_%s", emoji, prefix, codename)
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	username := genUsername(addr)

	mu.Lock()
	clients[conn] = username
	conn.Write([]byte("[USERNAME]" + username + "\n"))
	mu.Unlock()

	// ⬇️ Mark system message
	joinMsg := fmt.Sprintf("[SYSTEM] joined OMSAY server\n", username)
	broadcast(joinMsg, nil)
	fmt.Print(joinMsg)

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		//timestamp := time.Now().Format("15:04:05")
		formatted := fmt.Sprintf("%s : %s\n", username, msg)
		broadcast(formatted, conn)
		//broadcast(fmt.Sprintf("%s: %s\n", username, msg), conn)
	}

	mu.Lock()
	delete(clients, conn)
	mu.Unlock()

	// ⬇️ Mark system message
	leaveMsg := fmt.Sprintf("[SYSTEM] 🚪 %s left the chat\n", username)
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
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // interface down or loopback
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			if !ip.IsPrivate() {
				continue // ignore public IPs, prefer local
			}

			return ip.String()
		}
	}

	return "127.0.0.1"
}
