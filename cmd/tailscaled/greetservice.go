package main

import (
	"log"

	"github.com/gorilla/websocket"
)

type GreetService struct {
	c *websocket.Conn
}

func (g *GreetService) Greet(name string) string {
	return "Hellooo " + name + "!"
}

func (g *GreetService) Send(message string) {
	err := g.c.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("write:", err)
		return
	}
}
