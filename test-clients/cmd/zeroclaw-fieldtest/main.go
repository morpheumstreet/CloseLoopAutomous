// Zeroclaw field test: automated text ping/pong + optional WS control ping, transcript -> report file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/morpheumstreet/CloseLoopAutomous/test-clients/pkg/zeroclaw/chatws"
)

// Dashboard /ws/chat outbound shape (from ZeroClaw web client: JSON.stringify({type:"message",content})).
type chatMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func main() {
	host := flag.String("host", "51fbec927350dce6c86d5bd30e40e76a.ns.mormscan.io", "Zeroclaw host:port (no scheme)")
	token := flag.String("token", os.Getenv("ZEROCLAW_TOKEN"), "bearer token (or ZEROCLAW_TOKEN)")
	tls := flag.Bool("tls", true, "use wss://")
	outPath := flag.String("out", "zeroclaw-field-test-report.txt", "write transcript here")
	turnWait := flag.Duration("turn-wait", 45*time.Second, "max wait for inbound after each outbound chat message")
	overall := flag.Duration("overall", 3*time.Minute, "max total test duration after connect")
	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		log.Fatal("missing -token or ZEROCLAW_TOKEN")
	}

	rep := newReporter(*outPath)
	defer func() { _ = rep.flush() }()

	rep.writePrologue(*host, *tls)
	rep.logf("meta", "outbound chat frames use JSON {\"type\":\"message\",\"content\":...} (ZeroClaw UI wire format)")

	u := chatws.BuildURL(*host, *tls)
	rep.logf("meta", "dial %s", u)

	conn, resp, err := chatws.Dial(*host, *token, *tls)
	if err != nil {
		if resp != nil {
			rep.logf("meta", "HTTP %s", resp.Status)
		}
		rep.logf("error", "dial: %v", err)
		log.Fatal(err)
	}
	defer conn.Close()

	rep.logf("meta", "connected subprotocol=%q", conn.Subprotocol())

	conn.SetPingHandler(func(appData string) error {
		rep.logLine("IN", "ws-ping", string(appData))
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})
	conn.SetPongHandler(func(appData string) error {
		rep.logLine("IN", "ws-pong", string(appData))
		return nil
	})

	inbound := make(chan string, 64)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					rep.logf("error", "read closed: %v", err)
				}
				return
			}
			inbound <- string(message)
		}
	}()

	deadline := time.After(*overall)

	sendChat := func(content string) error {
		payload, err := json.Marshal(chatMessage{Type: "message", Content: content})
		if err != nil {
			return err
		}
		rep.logLine("OUT", "json", string(payload))
		return conn.WriteMessage(websocket.TextMessage, payload)
	}

	sendWSPing := func(payload string) error {
		rep.logLine("OUT", "ws-ping", payload)
		return conn.WriteControl(websocket.PingMessage, []byte(payload), time.Now().Add(5*time.Second))
	}

	drainInboundFor := func(d time.Duration) []string {
		var got []string
		until := time.NewTimer(d)
		defer until.Stop()
		for {
			select {
			case msg := <-inbound:
				rep.logLine("IN", "text", msg)
				got = append(got, msg)
			case <-until.C:
				return got
			case <-readDone:
				return got
			case <-deadline:
				return got
			}
		}
	}

	waitFirst := func(d time.Duration) (string, bool) {
		select {
		case msg := <-inbound:
			rep.logLine("IN", "text", msg)
			return msg, true
		case <-time.After(d):
			return "", false
		case <-readDone:
			return "", false
		case <-deadline:
			return "", false
		}
	}

	// 1) session / first server frame
	if _, ok := waitFirst(*turnWait); !ok {
		rep.logf("meta", "no inbound within %s (server may still be idle)", *turnWait)
	}

	// 2) WebSocket ping → expect pong (handled by SetPongHandler + read loop)
	if err := sendWSPing("fieldtest"); err != nil {
		rep.logf("error", "ws ping: %v", err)
	} else {
		drainInboundFor(2 * time.Second)
	}

	// 3) chat script (user-visible text; framed as type:message JSON)
	script := []string{
		"ping",
		"pong",
		"Field test (CloseLoopAutomous test-clients): reply with one short plain-text sentence so we can record it in a field-test report.",
	}
	for _, line := range script {
		select {
		case <-deadline:
			rep.logf("meta", "stopped: overall deadline")
			goto done
		case <-readDone:
			goto done
		default:
		}
		if err := sendChat(line); err != nil {
			rep.logf("error", "write: %v", err)
			goto done
		}
		drainInboundFor(*turnWait)
	}

	rep.logf("meta", "script complete; collecting late frames 3s")
	drainInboundFor(3 * time.Second)

done:
	_ = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "field test done"))

	rep.logf("meta", "report file: %s", *outPath)
	log.Printf("wrote %s", *outPath)
}

type reporter struct {
	path     string
	mu       sync.Mutex
	lines    []string
	prologue string
}

func newReporter(path string) *reporter {
	return &reporter{path: path}
}

func (r *reporter) writePrologue(host string, tls bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prologue = fmt.Sprintf(
		"Zeroclaw field test report\n"+
			"generated: %s\n"+
			"host: %s tls=%v\n"+
			"--- transcript ---\n",
		time.Now().UTC().Format(time.RFC3339), host, tls,
	)
}

func (r *reporter) logLine(dir, kind, body string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	r.lines = append(r.lines, fmt.Sprintf("[%s] %s %s %s", ts, dir, kind, body))
}

func (r *reporter) logf(kind, format string, args ...any) {
	r.logLine("meta", kind, fmt.Sprintf(format, args...))
}

func (r *reporter) flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	b.WriteString(r.prologue)
	for _, ln := range r.lines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteString("--- end ---\n")
	return os.WriteFile(r.path, []byte(b.String()), 0o644)
}
