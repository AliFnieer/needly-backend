package websocket

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AliFnieer/needly-backend/internal/config"
	"github.com/gin-gonic/gin"
	gorilla "github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// ServeWS upgrades the connection, registers the client, and starts pumps.
// The user must already be authenticated (auth middleware runs before this).
// If a household_id is provided, membership is verified.
func ServeWS(h *Hub, c *gin.Context, db *gorm.DB, cfg *config.Config) {
	conn, err := upgradeWithOriginCheck(c, cfg)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}

	// Get authenticated user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		_ = conn.WriteMessage(gorilla.CloseMessage,
			gorilla.FormatCloseMessage(gorilla.ClosePolicyViolation, "unauthenticated"))
		conn.Close()
		return
	}

	uid, ok := userID.(float64)
	if !ok {
		_ = conn.WriteMessage(gorilla.CloseMessage,
			gorilla.FormatCloseMessage(gorilla.ClosePolicyViolation, "invalid user id"))
		conn.Close()
		return
	}

	// Parse household id from path param and verify membership
	hidStr := c.Param("household_id")
	var hid uint
	if hidStr != "" {
		v, err := strconv.ParseUint(hidStr, 10, 64)
		if err != nil {
			_ = conn.WriteMessage(gorilla.CloseMessage,
				gorilla.FormatCloseMessage(gorilla.ClosePolicyViolation, "invalid household_id"))
			conn.Close()
			return
		}
		hid = uint(v)

		// Verify the user is a member of this household
		if err := verifyHouseholdMembership(db, hid, uint(uid)); err != nil {
			_ = conn.WriteMessage(gorilla.CloseMessage,
				gorilla.FormatCloseMessage(gorilla.ClosePolicyViolation, "not a household member"))
			conn.Close()
			return
		}
	}

	// Auto-generate client ID from user + household (no spoofable query param)
	clientID := strconv.FormatUint(uint64(uid), 10) + ":" + strconv.FormatUint(uint64(hid), 10)

	client := &Client{
		ID:          clientID,
		HouseholdID: hid,
		UserID:      uint(uid),
		Send:        make(chan []byte, 256),
	}

	h.Register(client)

	conn.SetPingHandler(func(string) error {
		return conn.WriteControl(gorilla.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
	})

	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(gorilla.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					conn.Close()
					return
				}
			case msg, ok := <-client.Send:
				if !ok {
					conn.Close()
					return
				}
				if err := conn.WriteMessage(gorilla.TextMessage, msg); err != nil {
					conn.Close()
					return
				}
			}
		}
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	h.Unregister(client)
	conn.Close()
}

// upgradeWithOriginCheck performs the WebSocket upgrade with origin validation.
func upgradeWithOriginCheck(c *gin.Context, cfg *config.Config) (*gorilla.Conn, error) {
	allowedOrigins := cfg.CORS.AllowedOrigins

	upgrader := gorilla.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser clients
			}
			// If no origins configured, allow all (same as CORS middleware)
			if len(allowedOrigins) == 0 {
				return true
			}
			for _, ao := range allowedOrigins {
				if ao == "*" || ao == origin {
					return true
				}
			}
			return false
		},
	}

	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

// verifyHouseholdMembership checks that the user belongs to the household.
func verifyHouseholdMembership(db *gorm.DB, householdID, userID uint) error {
	var count int64
	if err := db.Table("household_members").
		Where("household_id = ? AND user_id = ?", householdID, userID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("not a household member")
	}
	return nil
}
