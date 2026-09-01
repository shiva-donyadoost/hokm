// Package room implements the lobby domain: rooms, membership, readiness,
// host controls, and AI slot management. Rooms are transport-agnostic;
// the WebSocket layer subscribes to changes in Phase 6.
package room

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrRoomNotFound      = errors.New("room: room not found")
	ErrRoomFull          = errors.New("room: room is full")
	ErrAlreadyInRoom     = errors.New("room: user already in room")
	ErrNotInRoom         = errors.New("room: user is not in this room")
	ErrNotHost           = errors.New("room: only the host can do that")
	ErrGameInProgress    = errors.New("room: game already in progress")
	ErrInvalidName       = errors.New("room: invalid room name")
	ErrInvalidVisibility = errors.New("room: invalid visibility")
	ErrCannotKickSelf    = errors.New("room: host cannot kick themselves")
	ErrNoEmptySlot       = errors.New("room: no empty seat for an AI slot")
	ErrNotAnAI           = errors.New("room: member is not an AI")
	ErrCodeConflict      = errors.New("room: could not allocate room code")
	ErrInvalidRoundCount = errors.New("room: invalid round count")
	ErrInvalidGameSpeed  = errors.New("room: invalid game speed")
	ErrInvalidSeat       = errors.New("room: invalid seat")
	ErrEmptySeat         = errors.New("room: source seat is empty")
)

// RoomSettings carries creator-chosen match configuration.
type RoomSettings struct {
	RoundCount  int
	GameSpeed   string // fast | medium | slow
	ChatEnabled bool
}

// DefaultRoundCounts are the supported creator options; the list itself is
// configurable (see config.GameConfig.AllowedRoundCounts).
var DefaultRoundCounts = []int{1, 3, 5}

// ValidGameSpeeds are the supported speeds; timeouts come from config.
var ValidGameSpeeds = []string{"fast", "medium", "slow"}

// ValidateSettings checks creator input against the allowed option sets.
func ValidateSettings(s RoomSettings, allowedRounds []int) error {
	if s.RoundCount <= 0 {
		return ErrInvalidRoundCount
	}
	ok := false
	for _, r := range allowedRounds {
		if r == s.RoundCount {
			ok = true
			break
		}
	}
	if !ok {
		return ErrInvalidRoundCount
	}
	speedOK := false
	for _, sp := range ValidGameSpeeds {
		if sp == s.GameSpeed {
			speedOK = true
			break
		}
	}
	if !speedOK {
		return ErrInvalidGameSpeed
	}
	return nil
}

// Visibility controls who can find and join a room.
type Visibility string

const (
	Public  Visibility = "public"
	Private Visibility = "private"
)

func (v Visibility) Valid() bool { return v == Public || v == Private }

// Member is a seated participant (human or AI).
type Member struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Seat         int    `json:"seat"`
	Ready        bool   `json:"ready"`
	IsHost       bool   `json:"is_host"`
	IsAI         bool   `json:"is_ai"`
	AIDifficulty string `json:"ai_difficulty,omitempty"`
}

// Room is a lobby that becomes a game table.
type Room struct {
	ID         string     `json:"id"`
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Visibility Visibility `json:"visibility"`
	HostID     string     `json:"host_id"`
	Members    []Member   `json:"members"` // index == seat, empty seats absent
	Status     string     `json:"status"`  // lobby | in_game
	CreatedAt  time.Time  `json:"created_at"`
	// Match settings chosen by the room creator (impliment.md §5, §10, §31).
	RoundCount  int    `json:"round_count"` // rounds to win the match
	GameSpeed   string `json:"game_speed"`  // fast | medium | slow
	ChatEnabled bool   `json:"chat_enabled"`
}

// InSeat reports whether the user occupies a seat.
func (r *Room) InSeat(userID string) bool {
	for _, m := range r.Members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

// Host returns a defensive copy so callers cannot mutate manager state.
func (r *Room) clone() Room {
	cp := *r
	cp.Members = append([]Member(nil), r.Members...)
	return cp
}

// Notifier receives room snapshots after each mutation. The WS layer
// implements it in Phase 6 to fan out updates.
type Notifier interface {
	RoomUpdated(Room)
}

// Manager owns all rooms in this process.
type Manager struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	byCode   map[string]string
	notifier Notifier
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room), byCode: make(map[string]string)}
}

// SetNotifier registers the change fan-out hook.
func (m *Manager) SetNotifier(n Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifier = n
}

func (m *Manager) notify(r *Room) {
	if m.notifier != nil {
		m.notifier.RoomUpdated(r.clone())
	}
}

// Create makes a new room hosted by the given user.
func (m *Manager) Create(hostID, hostName, name string, visibility Visibility, settings RoomSettings) (Room, error) {
	name = strings.TrimSpace(name)
	if l := len(name); l < 2 || l > 40 {
		return Room{}, ErrInvalidName
	}
	if !visibility.Valid() {
		return Room{}, ErrInvalidVisibility
	}
	if err := ValidateSettings(settings, DefaultRoundCounts); err != nil {
		return Room{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	code, err := m.allocCodeLocked()
	if err != nil {
		return Room{}, err
	}
	r := &Room{
		ID:          newRoomID(),
		Code:        code,
		Name:        name,
		Visibility:  visibility,
		HostID:      hostID,
		Status:      "lobby",
		CreatedAt:   time.Now().UTC(),
		RoundCount:  settings.RoundCount,
		GameSpeed:   settings.GameSpeed,
		ChatEnabled: settings.ChatEnabled,
		Members: []Member{{
			UserID: hostID, Username: hostName, Seat: 0, Ready: true, IsHost: true,
		}},
	}
	m.rooms[r.ID] = r
	m.byCode[code] = r.ID
	m.notify(r)
	return r.clone(), nil
}

// Join seats the user in the room with the given code.
func (m *Manager) Join(code, userID, username string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byCode[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return m.joinLocked(m.rooms[id], userID, username)
}

// JoinByID joins via room id (e.g., invite links).
func (m *Manager) JoinByID(roomID, userID, username string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return m.joinLocked(r, userID, username)
}

func (m *Manager) joinLocked(r *Room, userID, username string) (Room, error) {
	if r.Status != "lobby" {
		return Room{}, ErrGameInProgress
	}
	if r.InSeat(userID) {
		return Room{}, ErrAlreadyInRoom
	}
	if len(r.Members) >= 4 {
		return Room{}, ErrRoomFull
	}
	seat := 0
	taken := map[int]bool{}
	for _, mem := range r.Members {
		taken[mem.Seat] = true
	}
	for taken[seat] {
		seat++
	}
	r.Members = append(r.Members, Member{
		UserID: userID, Username: username, Seat: seat, Ready: false,
	})
	m.notify(r)
	return r.clone(), nil
}

// Leave removes the user; the host role transfers to the next human member
// and empty rooms are deleted.
func (m *Manager) Leave(roomID, userID string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	idx := -1
	for i, mem := range r.Members {
		if mem.UserID == userID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Room{}, ErrNotInRoom
	}
	r.Members = append(r.Members[:idx], r.Members[idx+1:]...)
	if len(r.Members) == 0 {
		delete(m.rooms, r.ID)
		delete(m.byCode, r.Code)
		return Room{}, nil
	}
	if r.HostID == userID {
		// Host transfer: first human member in seat order.
		for i := range r.Members {
			if !r.Members[i].IsAI {
				r.Members[i].IsHost = true
				r.HostID = r.Members[i].UserID
				break
			}
		}
	}
	m.notify(r)
	return r.clone(), nil
}

// SetReady toggles a human member's ready flag.
func (m *Manager) SetReady(roomID, userID string, ready bool) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	found := false
	for i := range r.Members {
		if r.Members[i].UserID == userID {
			r.Members[i].Ready = ready
			found = true
			break
		}
	}
	if !found {
		return Room{}, ErrNotInRoom
	}
	m.notify(r)
	return r.clone(), nil
}

// Kick removes a member; only the host may kick, and only humans.
func (m *Manager) Kick(roomID, hostID, targetID string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	if r.HostID != hostID {
		return Room{}, ErrNotHost
	}
	if hostID == targetID {
		return Room{}, ErrCannotKickSelf
	}
	if !r.InSeat(targetID) {
		return Room{}, ErrNotInRoom
	}
	room, err := m.leaveLocked(r, targetID)
	if err != nil {
		return Room{}, err
	}
	return room, nil
}

func (m *Manager) leaveLocked(r *Room, userID string) (Room, error) {
	for i, mem := range r.Members {
		if mem.UserID == userID {
			r.Members = append(r.Members[:i], r.Members[i+1:]...)
			if len(r.Members) == 0 {
				delete(m.rooms, r.ID)
				delete(m.byCode, r.Code)
				return Room{}, nil
			}
			if r.HostID == userID {
				for j := range r.Members {
					if !r.Members[j].IsAI {
						r.Members[j].IsHost = true
						r.HostID = r.Members[j].UserID
						break
					}
				}
			}
			m.notify(r)
			return r.clone(), nil
		}
	}
	return Room{}, ErrNotInRoom
}

// AddAI fills the first empty seat with an AI member. Only the host may.
func (m *Manager) AddAI(roomID, hostID, difficulty, username string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	if r.HostID != hostID {
		return Room{}, ErrNotHost
	}
	if r.Status != "lobby" {
		return Room{}, ErrGameInProgress
	}
	if len(r.Members) >= 4 {
		return Room{}, ErrRoomFull
	}
	m.addAILocked(r, difficulty, username)
	m.notify(r)
	return r.clone(), nil
}

var aiDifficulties = []string{"easy", "medium", "hard", "expert", "pro"}

func randomAIDifficulty() string {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(aiDifficulties))))
	if err != nil {
		return "medium"
	}
	return aiDifficulties[n.Int64()]
}

func (m *Manager) addAILocked(r *Room, difficulty, username string) {
	seat := 0
	taken := map[int]bool{}
	for _, mem := range r.Members {
		taken[mem.Seat] = true
	}
	for taken[seat] {
		seat++
	}
	r.Members = append(r.Members, Member{
		UserID: fmt.Sprintf("ai:%s:%d", difficulty, seat), Username: username,
		Seat: seat, Ready: true, IsAI: true, AIDifficulty: difficulty,
	})
}

// FillEmptyWithAI seats a randomly-difficult AI in every empty seat.
// Only the host may call it, and only while the room is in lobby.
func (m *Manager) FillEmptyWithAI(roomID, hostID string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	if r.HostID != hostID {
		return Room{}, ErrNotHost
	}
	if r.Status != "lobby" {
		return Room{}, ErrGameInProgress
	}
	if len(r.Members) >= 4 {
		return Room{}, ErrNoEmptySlot
	}
	for len(r.Members) < 4 {
		d := randomAIDifficulty()
		m.addAILocked(r, d, "AI ("+d+")")
	}
	m.notify(r)
	return r.clone(), nil
}

// RemoveAI removes an AI member; only the host may.
func (m *Manager) RemoveAI(roomID, hostID, aiUserID string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	if r.HostID != hostID {
		return Room{}, ErrNotHost
	}
	for _, mem := range r.Members {
		if mem.UserID == aiUserID && !mem.IsAI {
			return Room{}, ErrNotAnAI
		}
	}
	return m.leaveLocked(r, aiUserID)
}

// Get returns a copy of the room.
func (m *Manager) Get(roomID string) (Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return r.clone(), nil
}

// ByCode resolves a join code to a room copy.
func (m *Manager) ByCode(code string) (Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byCode[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return m.rooms[id].clone(), nil
}

// ListPublic returns public lobby rooms sorted by creation time.
func (m *Manager) ListPublic() []Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		if r.Visibility == Public && r.Status == "lobby" {
			out = append(out, r.clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// MarkStarted flips the room into in_game status (called by the gameplay
// layer when a match starts).
func (m *Manager) MarkStarted(roomID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrRoomNotFound
	}
	r.Status = "in_game"
	m.notify(r)
	return nil
}

// MoveSeat lets the host move whoever occupies fromSeat onto toSeat
// (swap if occupied, slide if empty). Lobby only (ADR-0013).
func (m *Manager) MoveSeat(roomID, hostID string, fromSeat, toSeat int) (Room, error) {
	if fromSeat < 0 || fromSeat > 3 || toSeat < 0 || toSeat > 3 {
		return Room{}, ErrInvalidSeat
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	if r.HostID != hostID {
		return Room{}, ErrNotHost
	}
	if r.Status != "lobby" {
		return Room{}, ErrGameInProgress
	}
	if fromSeat == toSeat {
		return r.clone(), nil
	}
	var src, dst *Member
	for i := range r.Members {
		if r.Members[i].Seat == fromSeat {
			src = &r.Members[i]
		}
		if r.Members[i].Seat == toSeat {
			dst = &r.Members[i]
		}
	}
	if src == nil {
		return Room{}, ErrEmptySeat
	}
	if dst == nil {
		src.Seat = toSeat
	} else {
		src.Seat, dst.Seat = toSeat, fromSeat
	}
	m.notify(r)
	return r.clone(), nil
}

// Delete removes a lobby room. Host only; in-game rooms are rejected.
func (m *Manager) Delete(roomID, hostID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrRoomNotFound
	}
	if r.HostID != hostID {
		return ErrNotHost
	}
	if r.Status != "lobby" {
		return ErrGameInProgress
	}
	r.Status = "closed"
	m.notify(r)
	delete(m.rooms, r.ID)
	delete(m.byCode, r.Code)
	return nil
}

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (m *Manager) allocCodeLocked() (string, error) {
	for attempt := 0; attempt < 50; attempt++ {
		b := make([]byte, 6)
		for i := range b {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
			if err != nil {
				return "", fmt.Errorf("room: code rand: %w", err)
			}
			b[i] = codeAlphabet[n.Int64()]
		}
		code := string(b)
		if _, taken := m.byCode[code]; !taken {
			return code, nil
		}
	}
	return "", ErrCodeConflict
}

func newRoomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("r_%s", hexEncode(b))
}

func hexEncode(b []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexDigits[v>>4]
		out[i*2+1] = hexDigits[v&0x0f]
	}
	return string(out)
}
