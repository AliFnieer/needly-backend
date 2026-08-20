package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(id string, householdID, userID uint) *Client {
	return &Client{
		ID:          id,
		HouseholdID: householdID,
		UserID:      userID,
		Send:        make(chan []byte, 256),
	}
}

func TestRegisterAndUnregister(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	c := newTestClient("c1", 1, 1)
	hub.Register(c)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientCount())

	hub.Unregister(c)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, hub.GetClientCount())
}

func TestGetClientCount(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	assert.Equal(t, 0, hub.GetClientCount())

	c1 := newTestClient("c1", 1, 1)
	c2 := newTestClient("c2", 1, 2)
	c3 := newTestClient("c3", 2, 3)

	hub.Register(c1)
	hub.Register(c2)
	hub.Register(c3)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, hub.GetClientCount())

	hub.Unregister(c2)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, hub.GetClientCount())
}

func TestDeliverToHousehold(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	c1 := newTestClient("c1", 1, 1)
	c2 := newTestClient("c2", 1, 2)
	c3 := newTestClient("c3", 2, 3)

	hub.Register(c1)
	hub.Register(c2)
	hub.Register(c3)
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"type":"test"}`)
	hub.deliverToHousehold(1, msg)

	select {
	case got := <-c1.Send:
		assert.Equal(t, msg, got)
	case <-time.After(time.Second):
		t.Fatal("c1 did not receive message")
	}

	select {
	case got := <-c2.Send:
		assert.Equal(t, msg, got)
	case <-time.After(time.Second):
		t.Fatal("c2 did not receive message")
	}

	assert.Empty(t, c3.Send)
}

func TestDeliverToClient(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	c1 := newTestClient("c1", 1, 1)
	c2 := newTestClient("c2", 1, 2)

	hub.Register(c1)
	hub.Register(c2)
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"target":"c1"}`)
	hub.deliverToClient("c1", msg)

	select {
	case got := <-c1.Send:
		assert.Equal(t, msg, got)
	case <-time.After(time.Second):
		t.Fatal("c1 did not receive message")
	}

	assert.Empty(t, c2.Send)
}

func TestBroadcastToHousehold(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	c1 := newTestClient("c1", 5, 1)
	c2 := newTestClient("c2", 5, 2)
	c3 := newTestClient("c3", 6, 3)

	hub.Register(c1)
	hub.Register(c2)
	hub.Register(c3)
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"broadcast":true}`)
	hub.BroadcastToHousehold(5, msg)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		select {
		case got := <-c1.Send:
			assert.Equal(t, msg, got)
		case <-time.After(time.Second):
			t.Error("c1 did not receive broadcast")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case got := <-c2.Send:
			assert.Equal(t, msg, got)
		case <-time.After(time.Second):
			t.Error("c2 did not receive broadcast")
		}
	}()

	wg.Wait()
	assert.Empty(t, c3.Send)
}

func TestShutdown(t *testing.T) {
	hub := NewHub(nil)

	require.NotNil(t, hub.ctx)
	assert.NoError(t, hub.ctx.Err())

	hub.Shutdown()

	select {
	case <-hub.ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled after Shutdown")
	}
}

func TestUnregisterNonexistent(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	c := newTestClient("ghost", 1, 1)
	hub.Unregister(c)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, hub.GetClientCount())
}
