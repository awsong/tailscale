package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnlocal"
	"tailscale.com/safesocket"
	"tailscale.com/tailcfg"
)

//go:embed frontend/dist
var assets embed.FS

type Command struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

var maxClients = 2 // Maximum number of concurrent clients
type guiBackend struct {
	lb         *ipnlocal.LocalBackend
	socketPath string
	conns      map[*websocket.Conn]bool
	mu         *sync.Mutex
}

func NewGUIBackend(lb *ipnlocal.LocalBackend) *guiBackend {
	return &guiBackend{
		lb:         lb,
		socketPath: args.socketpath + "2",
		mu:         &sync.Mutex{},
	}
}

func RunGUIProxy() {
	dummyHost := "localhost:80"

	// Create a custom dialer using Unix socket or named pipe on Windows
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return safesocket.Connect(args.socketpath + "2")
		},
	}

	// Use a dummy URL with the correct protocol and the dummy host
	u := url.URL{Scheme: "ws", Host: dummyHost, Path: "/wails"}

	// Establish the WebSocket connection
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	app := application.New(application.Options{
		Name:        "MirageClient",
		Description: "Systray for MirageClient",
		Bind: []any{
			&GreetService{c: c},
		},
		Assets: application.AssetOptions{
			FS: assets,
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	systemTray := app.NewSystemTray()

	window := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Width:         700,
		Height:        650,
		Name:          "Systray Window",
		Frameless:     true,
		AlwaysOnTop:   true,
		Hidden:        true,
		DisableResize: true,
		ShouldClose: func(window *application.WebviewWindow) bool {
			window.Hide()
			return false
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})

	if runtime.GOOS == "darwin" {
		systemTray.SetTemplateIcon(icons.SystrayMacTemplate)
	}

	systemTray.AttachWindow(window).WindowOffset(5)

	// Start listening for messages
	go func() { //readMessages(c *websocket.Conn) {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			app.Events.Emit(&application.WailsEvent{
				Name: "onMessage",
				Data: message,
			})
		}
	}()

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin for simplicity
	},
}

func (g *guiBackend) commandHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP server connection to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	g.mu.Lock()
	if len(g.conns) >= maxClients {
		g.mu.Unlock()
		ws.WriteMessage(websocket.TextMessage, []byte("Max client limit reached. Connection refused."))
		return
	}
	g.conns[ws] = true
	g.mu.Unlock()

	log.Printf("New client connected\n")

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		log.Printf("Received message: %s\n", message)
		var command Command
		json.Unmarshal(message, &command)
		switch command.Cmd {
		case "init":
			{
				p := g.lb.Prefs()
				s := g.lb.State()
				nm := g.lb.NetMap()
				m, _ := json.Marshal(ipn.Notify{Prefs: &p, State: &s, NetMap: nm})
				if err := ws.WriteMessage(messageType, m); err != nil {
					log.Println("Error writing message:", err)
					break
				}
			}
		case "up":
			{
				state := g.lb.State()
				switch state {
				case ipn.Stopped:
					{
						g.lb.EditPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								WantRunning: true,
							},
							WantRunningSet: true,
						})
					}
				case ipn.NeedsLogin:
					{
						g.lb.StartLoginInteractive(r.Context())
					}
				}
			}
		case "down":
			{
				g.lb.EditPrefs(&ipn.MaskedPrefs{
					Prefs: ipn.Prefs{
						WantRunning: false,
					},
					WantRunningSet: true,
				})
			}
		case "exitNode":
			{
				g.lb.EditPrefs(&ipn.MaskedPrefs{
					Prefs: ipn.Prefs{
						ExitNodeID:             tailcfg.StableNodeID(command.Data),
						ExitNodeAllowLANAccess: true,
					},
					ExitNodeIDSet:             true,
					ExitNodeAllowLANAccessSet: true,
				})
			}
		case "logout":
			{
				g.lb.Logout(context.Background())
			}
		}
	}

	// Remove client from the map when they disconnect
	g.mu.Lock()
	delete(g.conns, ws)
	g.mu.Unlock()

	log.Printf("Client disconnected\n")
}
func (g *guiBackend) run() {

	g.lb.SetNotifyCallback(func(n ipn.Notify) {
		jsonBytes, err := json.Marshal(n)
		if err != nil {
			log.Fatalf("Error marshalling to JSON: %v", err)
		}
		for client := range g.conns {
			// The callback is called in a goroutine, so we need to lock the mutex
			g.mu.Lock()
			if err := client.WriteMessage(websocket.TextMessage, jsonBytes); err != nil {
				log.Println("Error writing message:", err)
			}
			g.mu.Unlock()
		}
	})

	ln, err := safesocket.Listen(g.socketPath)
	if err != nil {
		log.Fatalf("safesocket.Listen: %v", err)
	}

	http.HandleFunc("/wails", g.commandHandler)
	hs := &http.Server{}
	if err := hs.Serve(ln); err != nil {
		log.Fatal(err)
	}
	if err := hs.Serve(ln); err != nil {
		log.Fatalf("Failed to start guiBackend server: %v", err)
	}
}
