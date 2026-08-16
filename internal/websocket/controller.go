package websocket

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"
)

var upgrader = gorilla.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// ServeWS upgrades the connection, registers the client, and starts pumps.
func ServeWS(h *Hub, c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("websocket upgrade error: %v", err)
        return
    }

    // Parse household id from path param
    hidStr := c.Param("household_id")
    var hid uint
    if hidStr != "" {
        if v, err := strconv.ParseUint(hidStr, 10, 64); err == nil {
            hid = uint(v)
        }
    }

    // Client ID may be provided as query param `id`, otherwise generate one.
    clientID := c.Query("id")
    if clientID == "" {
        clientID = strconv.FormatInt(time.Now().UnixNano(), 10)
    }

    client := &Client{
        ID:          clientID,
        HouseholdID: hid,
        Send:        make(chan []byte, 256),
    }

    h.Register(client)

    // Start write pump
    go func() {
        for msg := range client.Send {
            if err := conn.WriteMessage(gorilla.TextMessage, msg); err != nil {
                break
            }
        }
        conn.Close()
    }()

    // Read loop: drain until error, then unregister
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            break
        }
    }

    h.Unregister(client)
    conn.Close()
}
