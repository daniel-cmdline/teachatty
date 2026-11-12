package main

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
)

type connectionManager struct {
	mu        sync.Mutex
	accepting bool
	forced    bool
	pending   int
	active    map[*websocket.Conn]struct{}
	changed   chan struct{}
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		accepting: true,
		active:    make(map[*websocket.Conn]struct{}),
		changed:   make(chan struct{}),
	}
}

func (m *connectionManager) signalLocked() {
	close(m.changed)
	m.changed = make(chan struct{})
}

type upgradeReservation struct {
	manager *connectionManager
	once    sync.Once
}

func (m *connectionManager) reserve() (*upgradeReservation, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.accepting {
		return nil, false
	}
	m.pending++
	m.signalLocked()
	return &upgradeReservation{manager: m}, true
}

// register moves the reservation into active atomically; after forced shutdown it rejects
// upgrades that were already in flight so none can escape the final snapshot.
func (r *upgradeReservation) register(conn *websocket.Conn) bool {
	registered := false
	r.once.Do(func() {
		m := r.manager
		m.mu.Lock()
		m.pending--
		if !m.forced {
			m.active[conn] = struct{}{}
			registered = true
		}
		m.signalLocked()
		m.mu.Unlock()
	})
	return registered
}

func (r *upgradeReservation) release() {
	r.once.Do(func() {
		m := r.manager
		m.mu.Lock()
		m.pending--
		m.signalLocked()
		m.mu.Unlock()
	})
}

func (m *connectionManager) stopAccepting() {
	m.mu.Lock()
	m.accepting = false
	m.signalLocked()
	m.mu.Unlock()
}

func (m *connectionManager) rejectPending() {
	m.mu.Lock()
	m.forced = true
	m.signalLocked()
	m.mu.Unlock()
}

func (m *connectionManager) waitPending(ctx context.Context) error {
	for {
		m.mu.Lock()
		if m.pending == 0 {
			m.mu.Unlock()
			return nil
		}
		changed := m.changed
		m.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// snapshot is safe after stopAccepting and waitPending: no untracked reservation can
// subsequently register unless forced shutdown rejects it.
func (m *connectionManager) snapshot() map[*websocket.Conn]struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	connections := make(map[*websocket.Conn]struct{}, len(m.active))
	for conn := range m.active {
		connections[conn] = struct{}{}
	}
	return connections
}

func (m *connectionManager) remove(conn *websocket.Conn) {
	m.mu.Lock()
	delete(m.active, conn)
	m.signalLocked()
	m.mu.Unlock()
}

func (m *connectionManager) waitEmpty(ctx context.Context) error {
	for {
		m.mu.Lock()
		if len(m.active) == 0 {
			m.mu.Unlock()
			return nil
		}
		changed := m.changed
		m.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *connectionManager) forceClose() {
	m.mu.Lock()
	m.forced = true
	connections := make([]*websocket.Conn, 0, len(m.active))
	for conn := range m.active {
		connections = append(connections, conn)
	}
	m.signalLocked()
	m.mu.Unlock()
	for _, conn := range connections {
		_ = conn.Close()
	}
}
