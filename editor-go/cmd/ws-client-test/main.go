// Temporary client to smoke-test the gode-engine WebSocket server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/gorilla/websocket"

	"gode/editor/engine"
)

func main() {
	port := flag.Int("port", 0, "engine port")
	flag.Parse()

	url := fmt.Sprintf("ws://127.0.0.1:%d", *port)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// 1. open document
	sendCmd(conn, engine.Command{Cmd: "open_document", Text: "hello world\nline two"})
	// 2. type a char
	sendCmd(conn, engine.Command{Cmd: "input", Type: "key", Key: &engine.InputKey{KeyType: "press", Key: "H", Rune: "!"}})
	// 3. read content
	sendCmd(conn, engine.Command{Cmd: "get_content", ID: 42})
	// 4. shutdown
	sendCmd(conn, engine.Command{Cmd: "shutdown"})

	for {
		var ev engine.Event
		if err := conn.ReadJSON(&ev); err != nil {
			log.Fatal(err)
		}
		switch ev.Evt {
		case "ready":
			fmt.Println("got ready")
		case "frame":
			fmt.Printf("got frame %dx%d (%d bytes) sel=%+v/%+v\n", ev.Width, ev.Height, len(ev.Data), ev.Anchor, ev.Active)
		case "content":
			fmt.Printf("content id=%d: %q\n", ev.ID, ev.Content)
		case "edited":
			fmt.Printf("edited %+v %q\n", ev.Range, ev.EditText)
		case "pong":
			fmt.Println("got pong")
		default:
			fmt.Println("got", ev.Evt)
		}
	}
}

func sendCmd(conn *websocket.Conn, cmd engine.Command) {
	data, _ := json.Marshal(cmd)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Fatal(err)
	}
}
