package main

/*
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin for simplicity
	},
}

// clients keeps track of the current connected clients
var clients = make(map[*websocket.Conn]bool)
var mutex = &sync.Mutex{} // Protects access to clients
var maxClients = 2        // Maximum number of concurrent clients
var lb *ipnlocal.LocalBackend

type Command struct {
	Cmd  string `json:"cmd"`
	Data string `json:"data"`
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP server connection to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	if len(clients) >= maxClients {
		mutex.Unlock()
		ws.WriteMessage(websocket.TextMessage, []byte("Max client limit reached. Connection refused."))
		return
	}
	clients[ws] = true
	mutex.Unlock()

	log.Printf("New client connected, total clients: %d\n", len(clients))

	// Echo received messages back to the client
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
				lb.Prefs()
				p := lb.Prefs()
				s := lb.State()
				nm := lb.NetMap()
				m, _ := json.Marshal(ipn.Notify{Prefs: &p, State: &s, NetMap: nm})
				if err := ws.WriteMessage(messageType, m); err != nil {
					log.Println("Error writing message:", err)
					break
				}
			}
		case "up":
			{
				state := lb.State()
				switch state {
				case ipn.Stopped:
					{
						lb.EditPrefs(&ipn.MaskedPrefs{
							Prefs: ipn.Prefs{
								WantRunning: true,
							},
							WantRunningSet: true,
						})
					}
				case ipn.NeedsLogin:
					{
						lb.StartLoginInteractive(r.Context())
					}
				}
			}
		case "down":
			{
				lb.EditPrefs(&ipn.MaskedPrefs{
					Prefs: ipn.Prefs{
						WantRunning: false,
					},
					WantRunningSet: true,
				})
			}
		case "exitNode":
			{
				lb.EditPrefs(&ipn.MaskedPrefs{
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
				lb.Logout(context.Background())
			}
		}
	}

	// Remove client from the map when they disconnect
	mutex.Lock()
	delete(clients, ws)
	mutex.Unlock()

	log.Printf("Client disconnected, total clients: %d\n", len(clients))
}

func runCustomAPIServer(localBackend *ipnlocal.LocalBackend) {

	lb = localBackend
	lb.SetNotifyCallback(func(n ipn.Notify) {
		jsonBytes, err := json.Marshal(n)
		if err != nil {
			log.Fatalf("Error marshalling to JSON: %v", err)
		}
		for client := range clients {
			// The callback is called in a goroutine, so we need to lock the mutex
			mutex.Lock()
			if err := client.WriteMessage(websocket.TextMessage, jsonBytes); err != nil {
				log.Println("Error writing message:", err)
			}
			mutex.Unlock()
		}
	})
	http.HandleFunc("/echo", commandHandler)

	log.Println("Starting echo server on :8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

*/
