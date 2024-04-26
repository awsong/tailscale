package main

import (
	"embed"
	_ "embed"
	"log"
	"net"
	"net/url"
	"runtime"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

//go:embed frontend/dist
var assets embed.FS

func ui() {
	// Define the Unix socket path and the dummy host for the WebSocket connection
	socketPath := "/tmp/echo.sock"
	dummyHost := "localhost:80"

	// Create a custom dialer using the Unix socket
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	// Use a dummy URL with the correct protocol and the dummy host
	u := url.URL{Scheme: "ws", Host: dummyHost, Path: "/echo"}

	// Establish the WebSocket connection
	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	app := application.New(application.Options{
		Name:        "Systray Demo",
		Description: "A demo of the Systray API",
		Bind: []any{
			&GreetService{c: c},
		},
		//Assets:      application.AlphaAssets,
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
		Name:          "Systray Demo Window",
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
		KeyBindings: map[string]func(window *application.WebviewWindow){
			"F12": func(window *application.WebviewWindow) {
				systemTray.OpenMenu()
			},
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
			log.Printf("Received: %s", message)
			app.Events.Emit(&application.WailsEvent{
				Name: "onMessage",
				Data: message,
			})
		}
	}()

	// Send a message to the server
	message := "Hello, WebSocket Server via Unix Socket!"
	log.Printf("Sending: %s", message)
	err = c.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("write:", err)
		return
	}

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
