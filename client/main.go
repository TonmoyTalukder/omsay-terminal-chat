package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
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

const currentVersion = "v25.5.6.3"

const updateURL = "https://github.com/TonmoyTalukder/omsay-terminal-chat/releases/latest/download/omsay.exe"

func updateExecutable(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Download to a temp file
	tmpFile := "omsay_new.exe"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Replace current binary
	currExe, _ := os.Executable()
	err = os.Rename(tmpFile, currExe)
	return err
}

func checkForUpdate() {
	resp, err := http.Get("https://raw.githubusercontent.com/TonmoyTalukder/omsay-terminal-chat/main/client/version.txt")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	latest, _ := io.ReadAll(resp.Body)
	latestVersion := strings.TrimSpace(string(latest))

	if latestVersion != currentVersion {
		color.Yellow("🚀 New version available: %s (you have %s)", latestVersion, currentVersion)
		fmt.Print("⚙️  Updating OMSAY... ")

		err := updateExecutable(updateURL)
		if err != nil {
			color.Red("❌ Update failed: %v", err)
			return
		}

		color.Green("✅ Updated successfully! Restarting...\n")
		exec.Command("omsay.exe").Start()
		os.Exit(0)
	}
}

func main() {
	clearTerminal()
	checkForUpdate()
	printHeader()
	showLoading("Starting OMSAY Chat Server", 2*time.Second)

	// Speaker init once
	speaker.Init(beep.SampleRate(44100), 44100/10) // standard rate

	serverAddr := discoverServer()
	conn, err := net.Dial("tcp", serverAddr+":8000")
	if err != nil {
		color.Red("Could not connect to server: %v", err)
		return
	}
	defer conn.Close()

	//playSound("assets/connect.wav")
	playEmbeddedSound(connectSound)
	color.HiGreen("\nConnected to OMSAY server at [%s]", serverAddr)
	fmt.Println()

	go readMessages(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if myUsername != "" {
			fmt.Print(color.HiBlueString(myUsername + " » "))
		} else {
			fmt.Print(color.HiBlueString("You » "))
		}

		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		if strings.TrimSpace(text) != "" {
			conn.Write([]byte(text + "\n"))
		}
	}
}

func printHeader() {
	fmt.Print("\033[38;2;244;120;48m")
	fmt.Println("")
	fmt.Println("╔════════════════════════════════════╗")
	fmt.Println("║           WELCOME TO OMSAY         ║")
	fmt.Println("╚════════════════════════════════════╝")
	fmt.Println("")
	fmt.Print("\033[0m")
}

func discoverServer() string {
	addr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:9000")
	conn, _ := net.DialUDP("udp", nil, addr)
	conn.Write([]byte("DISCOVER"))
	conn.Close()

	listenAddr, _ := net.ResolveUDPAddr("udp", ":9001")
	udpConn, _ := net.ListenUDP("udp", listenAddr)
	defer udpConn.Close()

	udpConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 1024)
	n, _, err := udpConn.ReadFromUDP(buf)
	if err != nil {
		color.Yellow("⚠️  No server discovered in LAN. Please enter the IP address of the OMSAY server:")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			color.Red("❌ No server provided. Exiting.")
			os.Exit(1) // or return ""
		}
		return input
	}

	return strings.TrimSpace(string(buf[:n]))
}

func getLocalIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

func readMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			color.Red("\nDisconnected from OMSAY server.")
			return
		}

		msg = strings.TrimSpace(msg)

		if strings.HasPrefix(msg, "[USERNAME]") {
			myUsername = strings.TrimPrefix(msg, "[USERNAME]")
			continue // don't print
		}

		timestamp := time.Now().Format("15:04:05")

		// Style differently if it's my own message
		if strings.Contains(msg, myUsername) {
			styledMsg := fmt.Sprintf("[%s] %s", color.HiBlackString(timestamp), color.HiGreenString(msg))
			typeWriter(styledMsg, 2*time.Millisecond)
		} else {
			styledMsg := fmt.Sprintf("[%s] %s", color.HiBlackString(timestamp), color.HiMagentaString(msg))
			typeWriter(styledMsg, 2*time.Millisecond)
		}

		playEmbeddedSound(messageSound)
		showNotification("OMSAY", msg)
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

//func playSound(filePath string) {
//	f, err := os.Open(filePath)
//	if err != nil {
//		return
//	}
//	defer f.Close()
//
//	streamer, format, err := wav.Decode(f)
//	if err != nil {
//		return
//	}
//	defer streamer.Close()
//
//	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
//	done := make(chan bool)
//	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
//		done <- true
//	})))
//	<-done
//}

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
