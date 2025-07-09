package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sshfun/keys"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	defaultPort = "2222"
	webPort     = "8080"
)

// Global client tracking
var (
	clientCount    int
	clientMutex    sync.RWMutex
	clientSessions = make(map[string]*ClientSession)
	sessionHistory = make([]*HistoricalSession, 0)
)

// ClientSession represents an active SSH tarpit session
type ClientSession struct {
	ID          string
	RemoteAddr  string
	StartTime   time.Time
	LinesSent   int
	LastMessage time.Time
}

// HistoricalSession represents a completed SSH tarpit session
type HistoricalSession struct {
	ID         string
	RemoteAddr string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	LinesSent  int
}

func main() {
	// Generate a private key for the SSH server
	privateKey, err := keys.LoadOrGenerateHostKey(".keystore/sshHostKey.private")
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	// Configure the SSH server
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	// Add the host key to the server config
	config.AddHostKey(privateKey)

	// Start the web server for monitoring
	go startWebServer()

	// Start listening on the SSH port
	port := defaultPort
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen on port ", port, ": ", err)
	}
	defer listener.Close()

	log.Printf("SSH tarpit server listening on port %s", port)
	log.Printf("Web monitoring interface available at http://localhost:%s", webPort)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	// Create a unique session ID for this client
	sessionID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())

	// Add client to tracking
	addClient(sessionID, conn.RemoteAddr().String())
	defer removeClient(sessionID)

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake from %s: %v", conn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (Session: %s)", conn.RemoteAddr(), sessionID)

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channel requests
	for newChannel := range chans {
		go handleChannel(newChannel, sessionID)
	}
}

func handleChannel(newChannel ssh.NewChannel, sessionID string) {
	// Only accept session channels
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel: %v", err)
		return
	}
	defer channel.Close()

	// Handle requests for this channel
	go handleRequests(requests, channel)

	// Clear screen and reset cursor position (like htop)
	channel.Write([]byte("\033[2J\033[1;1H\033[?25h"))

	// Simple command loop
	handleShell(channel, sessionID)
}

func handleRequests(requests <-chan *ssh.Request, channel ssh.Channel) {
	for req := range requests {
		switch req.Type {
		case "shell":
			// We'll handle the shell in the main channel handler
			req.Reply(true, nil)
		case "pty-req":
			// Handle PTY request
			req.Reply(true, nil)
		case "window-change":
			// Handle window resize
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

func handleShell(channel ssh.Channel, sessionID string) {
	// Initialize random seed based on current time
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Configuration for the tarpit
	const (
		minDelay      = 5000  // 5 seconds minimum delay
		maxDelay      = 15000 // 15 seconds maximum delay
		minLineLength = 10
		maxLineLength = 80
	)

	log.Printf("Starting SSH tarpit session for %s", sessionID)

	// Keep sending random banner lines indefinitely
	for {
		// Generate a random delay between messages (in milliseconds)
		delayMs := minDelay + rng.Intn(maxDelay-minDelay)

		// Sleep for the delay period
		time.Sleep(time.Duration(delayMs) * time.Millisecond)

		// Generate a random banner line
		line := generateRandomBannerLine(rng, minLineLength, maxLineLength)

		// Try to write the line to the channel
		_, err := channel.Write([]byte(line + "\r\n"))
		if err != nil {
			log.Printf("Client %s disconnected: %v", sessionID, err)
			return
		}

		// Update client statistics
		updateClientStats(sessionID)

		log.Printf("Sent tarpit line to %s: %s", sessionID, line)
	}
}

// generateRandomBannerLine creates a random SSH banner-like line
func generateRandomBannerLine(rng *rand.Rand, minLen, maxLen int) string {
	// Random length between min and max
	length := minLen + rng.Intn(maxLen-minLen)

	// Character set for random lines (printable ASCII excluding some problematic chars)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-_"

	line := make([]byte, length)
	for i := 0; i < length; i++ {
		line[i] = chars[rng.Intn(len(chars))]
	}

	// Make sure it doesn't start with "SSH-" as that would be a valid SSH banner
	result := string(line)
	if len(result) >= 4 && result[:4] == "SSH-" {
		result = "XXX-" + result[4:]
	}

	return result
}

// Client tracking functions
func addClient(sessionID, remoteAddr string) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	clientSessions[sessionID] = &ClientSession{
		ID:          sessionID,
		RemoteAddr:  remoteAddr,
		StartTime:   time.Now(),
		LinesSent:   0,
		LastMessage: time.Now(),
	}
	clientCount++
	log.Printf("Client added: %s (Total: %d)", sessionID, clientCount)
}

func removeClient(sessionID string) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if session, exists := clientSessions[sessionID]; exists {
		endTime := time.Now()
		duration := endTime.Sub(session.StartTime)

		// Add to historical sessions
		historical := &HistoricalSession{
			ID:         session.ID,
			RemoteAddr: session.RemoteAddr,
			StartTime:  session.StartTime,
			EndTime:    endTime,
			Duration:   duration,
			LinesSent:  session.LinesSent,
		}

		// Add to history and keep only top 50 longest sessions
		sessionHistory = append(sessionHistory, historical)

		// Sort by duration (longest first) and keep top 50
		if len(sessionHistory) > 1 {
			// Simple bubble sort for the new entry (more efficient for single additions)
			for i := len(sessionHistory) - 1; i > 0; i-- {
				if sessionHistory[i].Duration > sessionHistory[i-1].Duration {
					sessionHistory[i], sessionHistory[i-1] = sessionHistory[i-1], sessionHistory[i]
				} else {
					break
				}
			}
		}

		// Keep only top 50
		if len(sessionHistory) > 50 {
			sessionHistory = sessionHistory[:50]
		}

		log.Printf("Client removed: %s (Duration: %v, Lines sent: %d)",
			sessionID, duration, session.LinesSent)
		delete(clientSessions, sessionID)
		clientCount--
	}
}

func updateClientStats(sessionID string) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if session, exists := clientSessions[sessionID]; exists {
		session.LinesSent++
		session.LastMessage = time.Now()
	}
}

func getClientStats() (int, map[string]*ClientSession, []*HistoricalSession) {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	// Create a copy of the sessions map to avoid race conditions
	sessionsCopy := make(map[string]*ClientSession)
	for k, v := range clientSessions {
		sessionsCopy[k] = &ClientSession{
			ID:          v.ID,
			RemoteAddr:  v.RemoteAddr,
			StartTime:   v.StartTime,
			LinesSent:   v.LinesSent,
			LastMessage: v.LastMessage,
		}
	}

	// Create a copy of the history
	historyCopy := make([]*HistoricalSession, len(sessionHistory))
	for i, v := range sessionHistory {
		historyCopy[i] = &HistoricalSession{
			ID:         v.ID,
			RemoteAddr: v.RemoteAddr,
			StartTime:  v.StartTime,
			EndTime:    v.EndTime,
			Duration:   v.Duration,
			LinesSent:  v.LinesSent,
		}
	}

	return clientCount, sessionsCopy, historyCopy
}

// Web server implementation
func startWebServer() {
	http.HandleFunc("/", handleWebRoot)
	http.HandleFunc("/api/stats", handleAPIStats)

	log.Printf("Starting web server on port %s", webPort)
	if err := http.ListenAndServe(":"+webPort, nil); err != nil {
		log.Printf("Web server failed: %v", err)
	}
}

func handleWebRoot(w http.ResponseWriter, r *http.Request) {
	count, sessions, history := getClientStats()

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>SSH Tarpit Monitor</title>
    <meta http-equiv="refresh" content="5">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f0f0f0; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .stats { font-size: 24px; color: #2c5aa0; font-weight: bold; }
        table { border-collapse: collapse; width: 100%; margin-bottom: 30px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .duration { color: #666; }
        .lines { color: #0a7c42; font-weight: bold; }
        .leaderboard { margin-top: 30px; }
        .rank { font-weight: bold; color: #d4af37; }
        .rank.gold { color: #ffd700; }
        .rank.silver { color: #c0c0c0; }
        .rank.bronze { color: #cd7f32; }
        .section-title { color: #2c5aa0; border-bottom: 2px solid #2c5aa0; padding-bottom: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>SSH Tarpit Monitor</h1>
        <div class="stats">Currently trapped clients: {{.Count}}</div>
    </div>
    
    {{if .Sessions}}
    <h2 class="section-title">Active Sessions</h2>
    <table>
        <tr>
            <th>Session ID</th>
            <th>Remote Address</th>
            <th>Duration</th>
            <th>Lines Sent</th>
            <th>Last Message</th>
        </tr>
        {{range .Sessions}}
        <tr>
            <td>{{.ID}}</td>
            <td>{{.RemoteAddr}}</td>
            <td class="duration">{{.Duration}}</td>
            <td class="lines">{{.LinesSent}}</td>
            <td>{{.LastMessageFormatted}}</td>
        </tr>
        {{end}}
    </table>
    {{else}}
    <p>No active sessions.</p>
    {{end}}
    
    <div class="leaderboard">
        <h2 class="section-title">Leaderboard - Longest Trapped Sessions</h2>
        {{if .History}}
        <table>
            <tr>
                <th>Rank</th>
                <th>Remote Address</th>
                <th>Duration</th>
                <th>Lines Sent</th>
                <th>Date Trapped</th>
            </tr>
            {{range $index, $session := .History}}
            <tr>
                <td class="rank {{if eq $index 0}}gold{{else if eq $index 1}}silver{{else if eq $index 2}}bronze{{end}}">
                    {{if eq $index 0}}🥇{{else if eq $index 1}}🥈{{else if eq $index 2}}🥉{{else}}{{add $index 1}}{{end}}
                </td>
                <td>{{$session.RemoteAddr}}</td>
                <td class="duration">{{$session.DurationFormatted}}</td>
                <td class="lines">{{$session.LinesSent}}</td>
                <td>{{$session.StartTimeFormatted}}</td>
            </tr>
            {{end}}
        </table>
        {{else}}
        <p>No completed sessions yet. Trap some SSH clients to see the leaderboard!</p>
        {{end}}
    </div>
    
    <p><small>Page auto-refreshes every 5 seconds</small></p>
</body>
</html>`

	t := template.New("monitor").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})
	t, err := t.Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare data for template
	type sessionDisplay struct {
		ID                   string
		RemoteAddr           string
		Duration             string
		LinesSent            int
		LastMessageFormatted string
	}

	type historicalDisplay struct {
		RemoteAddr         string
		DurationFormatted  string
		LinesSent          int
		StartTimeFormatted string
	}

	var sessionList []sessionDisplay
	for _, session := range sessions {
		sessionList = append(sessionList, sessionDisplay{
			ID:                   session.ID,
			RemoteAddr:           session.RemoteAddr,
			Duration:             time.Since(session.StartTime).Round(time.Second).String(),
			LinesSent:            session.LinesSent,
			LastMessageFormatted: session.LastMessage.Format("15:04:05"),
		})
	}

	var historyList []historicalDisplay
	for _, session := range history {
		historyList = append(historyList, historicalDisplay{
			RemoteAddr:         session.RemoteAddr,
			DurationFormatted:  session.Duration.Round(time.Second).String(),
			LinesSent:          session.LinesSent,
			StartTimeFormatted: session.StartTime.Format("2006-01-02 15:04"),
		})
	}

	data := struct {
		Count    int
		Sessions []sessionDisplay
		History  []historicalDisplay
	}{
		Count:    count,
		Sessions: sessionList,
		History:  historyList,
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func handleAPIStats(w http.ResponseWriter, r *http.Request) {
	count, sessions, history := getClientStats()

	// Simple JSON response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"client_count": %d, "sessions": %d, "leaderboard_entries": %d}`,
		count, len(sessions), len(history))
}
