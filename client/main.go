// OMSAY Client
package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"golang.org/x/term"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/fatih/color"
	"github.com/go-toast/toast"
)

//go:embed internal/assets/connect.wav
var connectSound []byte

//go:embed internal/assets/message.wav
var messageSound []byte

var myUsername string

const currentVersion = "v25.5.10.3"

const updateURL = "https://github.com/TonmoyTalukder/omsay-terminal-chat/releases/latest/download/omsay.exe"

func updateExecutable(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the new version to a temp file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "omsay_new.exe")
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Launch the updater to replace the current exe
	currExe, _ := os.Executable()
	updaterPath := filepath.Join(filepath.Dir(currExe), "omsay-updater.exe")
	err = exec.Command(updaterPath, tmpFile, currExe).Start()
	if err != nil {
		return err
	}

	fmt.Println("🔁 Restarting via updater...")
	time.Sleep(500 * time.Millisecond)
	os.Exit(0) // Exit current app

	return nil
}

func checkForUpdate() {
	resp, err := http.Get("https://raw.githubusercontent.com/TonmoyTalukder/omsay-terminal-chat/main/client/version.txt")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	latest, _ := io.ReadAll(resp.Body)
	latestVersion := strings.TrimSpace(string(latest))

	if latestVersion == currentVersion {
		// No update needed
		return
	}

	color.Yellow("🚀 New version available: %s (you have %s)", latestVersion, currentVersion)
	fmt.Print("⚙️  Updating OMSAY... ")

	err = updateExecutable(updateURL)
	if err != nil {
		color.Red("❌ Update failed: %v", err)
		return
	}

	color.Green("✅ Updated successfully! Restarting...\n")
	exec.Command("omsay.exe").Start()
	os.Exit(0)
}

func main() {
	clearTerminal()
	checkForUpdate()
	printHeader()
	showLoading("MEET CREATOR: TONMOY TALUKDER", 1*time.Second)
	showLoading("Starting OMSAY Chat Server", 2*time.Second)

	// Speaker init once
	speaker.Init(beep.SampleRate(44100), 44100/10) // standard rate

	// Discover or start server
	serverAddr := discoverServer()

	// Establish a connection to the discovered or locally started server and Retry dialing up to 5 times
	var conn net.Conn
	var err error
	for i := 0; i < 5; i++ {
		color.Yellow("🔌 Attempting to connect to %s:8000...", serverAddr)
		conn, err = net.DialTimeout("tcp", serverAddr+":8000", 2*time.Second)
		if err != nil {
			color.Red("❌ Could not connect to server: %v", err)
			time.Sleep(1 * time.Second) // optional: wait before retry
			continue                    // 🔁 Retry again
		}
		color.Green("✅ Connected. Entering chat mode...")
		time.Sleep(1 * time.Second)
		break // ✅ Only break on success
	}
	if err != nil {
		color.Red("❌ Server didn't respond after multiple attempts.")
		color.Red("❌ Could not connect to server at [%s]: %v", serverAddr, err)
		fmt.Println("Please try restarting OMSAY or check your network.")
		return
	}

	defer conn.Close()

	// Play a sound indicating connection is made
	playEmbeddedSound(connectSound)
	color.HiGreen("\n✅ Connected to OMSAY server at [%s]", serverAddr)
	fmt.Println()

	// Start reading messages in the background
	go readMessages(conn)

	// Start chat interface
	showTypingPrompt()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Wait for user input
		if !scanner.Scan() {
			break
		}

		// Read user input text
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			// Send the typed text to the server
			//conn.Write([]byte(text + "\n"))
			conn.Write([]byte(strings.TrimSpace(text) + "\n"))
		} else {
			// Handle empty input
			typeWriter("...", 10*time.Millisecond)
		}

		// Keep showing the typing prompt
		showTypingPrompt()
	}
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(syscall.Stdout))
	if err != nil {
		return 80 // fallback
	}
	return width
}

func printHeader() {
	width := getTerminalWidth()
	title := " WELCOME TO OMSAY "
	border := "═"
	side := "║"

	// Ensure center title
	padding := (width - len(title) - 2) / 2 // -2 for side bars
	top := "╔" + strings.Repeat(border, width-2) + "╗"
	mid := side + strings.Repeat(" ", padding) + title + strings.Repeat(" ", width-2-len(title)-padding) + side
	bot := "╚" + strings.Repeat(border, width-2) + "╝"

	fmt.Print("\033[38;2;244;120;48m") // orange color
	fmt.Println(top)
	fmt.Println(mid)
	fmt.Println(bot)
	fmt.Print("\033[0m")
}

func discoverServer() string {
	fmt.Println("🔍 Scanning for OMSAY servers on your LAN...")

	addr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:9000")
	conn, _ := net.DialUDP("udp", nil, addr)
	conn.Write([]byte("DISCOVER"))
	conn.Close()

	listenAddr, _ := net.ResolveUDPAddr("udp", ":9001")
	udpConn, _ := net.ListenUDP("udp", listenAddr)
	defer udpConn.Close()

	udpConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Collect all responses
	serverIPs := []string{}
	seen := make(map[string]bool)

	buf := make([]byte, 1024)
	for {
		n, _, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		ip := strings.TrimSpace(string(buf[:n]))
		if !seen[ip] {
			seen[ip] = true
			serverIPs = append(serverIPs, ip)
		}
	}

	// If servers found
	if len(serverIPs) > 0 {
		// Remove our own IP from list if we're already hosting
		localIP := getLocalIP()
		isLocalRunning := isServerRunningLocally()

		filteredIPs := []string{}
		for _, ip := range serverIPs {
			if ip != localIP {
				filteredIPs = append(filteredIPs, ip)
			}
		}

		if isLocalRunning {
			fmt.Printf("✅ OMSAY server already running on your machine at %s. Joining it.\n", localIP)
			return localIP
		}

		if len(filteredIPs) == 0 {
			// Only our server is found
			fmt.Printf("✅ Detected only your OMSAY server at %s. Joining it.\n", localIP)
			return localIP
		}

		// Multiple remote servers found
		fmt.Println("🌐 Discovered OMSAY servers on LAN:")
		for i, ip := range filteredIPs {
			fmt.Printf("  [%d] %s\n", i+1, ip)
		}
		fmt.Printf("  [0] Start your own server\n")
		fmt.Print("Choose server [0]: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		if choice == "" || choice == "0" {
			if isLocalRunning {
				fmt.Println("⚠️ Server already running locally. Connecting instead of starting.")
				return localIP
			}
			startLocalServer()
			return localIP
		}
		idx := 0
		fmt.Sscanf(choice, "%d", &idx)
		if idx >= 1 && idx <= len(filteredIPs) {
			return filteredIPs[idx-1]
		}
		fmt.Println("⚠️ Invalid choice. Defaulting to your own server.")
		if isLocalRunning {
			return localIP
		}
		startLocalServer()
		return localIP
	}

	// No servers found
	fmt.Println("⚠️ No OMSAY servers discovered.")
	fmt.Print("Start your own server? (Y/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())
	if answer == "" || strings.ToLower(answer) == "y" {
		startLocalServer()
		time.Sleep(2 * time.Second) // ⏱️ Give it a second to bind to the port
		return getLocalIP()
	}

	fmt.Print("Enter OMSAY server IP: ")
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func isServerRunningLocally() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8000", time.Millisecond*500)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func getExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

func startLocalServer() {
	exeDir := getExecutableDir()
	serverPath := filepath.Join(exeDir, "omsay-server.exe")

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		color.Red("❌ omsay-server.exe not found. Please reinstall or check your installation.")
		os.Exit(1)
	}

	// Open in new terminal window
	cmd := exec.Command("cmd", "/C", "start", serverPath)
	err := cmd.Start()
	if err != nil {
		color.Red("❌ Failed to launch omsay-server.exe: %v", err)
		os.Exit(1)
	}

	// Give server time to boot up
	//time.Sleep(2 * time.Second)
	for i := 0; i < 5; i++ {
		if isServerRunningLocally() {
			break
		}
		time.Sleep(1 * time.Second)
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
				continue // not an IPv4 address
			}

			// Reject link-local addresses (169.254.x.x)
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}

			// Accept private IPs
			if ip.IsPrivate() {
				return ip.String()
			}
		}
	}

	return "127.0.0.1"
}

func readMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			color.Red("\n❌ Disconnected from OMSAY server: %v", err)
			return
		}
		msg = strings.TrimSpace(msg)

		if msg == "" {
			showTypingPrompt()
			continue
		}

		// Move cursor up and clear the current line
		fmt.Print("\033[2K\r")

		// Handle username assignment from server
		if strings.HasPrefix(msg, "[USERNAME]") {
			myUsername = strings.TrimPrefix(msg, "[USERNAME]")
			fmt.Printf("✓ Assigned username: %s\n", color.HiMagentaString(myUsername))
			showTypingPrompt()
			continue
		}

		// Handle self message (server echoes our message back for confirmation)
		if strings.HasPrefix(msg, "[SELF]") {
			//message := strings.TrimPrefix(msg, "[SELF]")
			//fmt.Printf("[%s] 📡 %s : %s\n",
			//	color.HiBlackString(time.Now().Format("15:04:05")),
			//	color.CyanString(myUsername),
			//	strings.TrimSpace(message))
			showTypingPrompt()
			continue
		}

		// Handle system messages
		if strings.HasPrefix(msg, "[SYSTEM]") {
			systemMessage := strings.TrimPrefix(msg, "[SYSTEM]")
			fmt.Printf("[%s] 📡 %s\n",
				color.HiBlackString(time.Now().Format("15:04:05")),
				color.YellowString(systemMessage))
			showTypingPrompt()
			fmt.Print("\033[2K\r")
			continue
		} else {
			parts := strings.SplitN(msg, "|", 2)
			if len(parts) == 2 {
				username := strings.TrimSpace(parts[0])
				message := strings.TrimSpace(parts[1])

				if username == myUsername {
					continue
				}

				fmt.Printf("[%s] 📡 %s : %s\n",
					color.HiBlackString(time.Now().Format("15:04:05")),
					color.CyanString(username),
					message)

				showTypingPrompt()
				playEmbeddedSound(messageSound)
				showNotification("OMSAY", fmt.Sprintf("%s: %s", username, message))
			}
		}
	}
}

func showNotification(title, message string) {
	notif := &toast.Notification{
		AppID:   "OMSAY Chat",
		Title:   title,
		Message: message,
	}
	notif.Push()
}

func showLoading(msg string, duration time.Duration) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + msg
	s.Start()
	time.Sleep(duration)
	s.Stop()
}

func clearTerminal() {
	print("\033[H\033[2J")
}

func typeWriter(text string, delay time.Duration) {
	for _, ch := range text {
		fmt.Printf("%c", ch)
		time.Sleep(delay)
	}
	fmt.Print(" ") // trailing space
	// Simulate blinking cursor
	for i := 0; i < 3; i++ {
		fmt.Print("_")
		time.Sleep(150 * time.Millisecond)
		fmt.Print("\b \b")
		time.Sleep(150 * time.Millisecond)
	}
	fmt.Println()
}

func playEmbeddedSound(data []byte) {
	streamer, _, err := wav.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	defer streamer.Close()

	//speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))
	<-done
}

func extractUsername(msg string) string {
	parts := strings.SplitN(msg, ":", 2)
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return "Unknown"
}

func extractMessageBody(msg string) string {
	parts := strings.SplitN(msg, ":", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return msg
}

func showTypingPrompt() {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] 📡 %s : ", color.HiBlackString(timestamp), color.HiBlueString(myUsername))
}
