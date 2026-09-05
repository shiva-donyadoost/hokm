package room

import (
	"errors"
	"testing"
)

func newTestManager() *Manager { return NewManager() }

func defaultSettings() RoomSettings {
	return RoomSettings{RoundCount: 1, GameSpeed: "medium", ChatEnabled: true}
}

func TestCreateAndCodeLookup(t *testing.T) {
	m := newTestManager()
	r, err := m.Create("u1", "Alice", "", "", "Friday Night", Public, defaultSettings())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(r.Code) != 6 {
		t.Fatalf("code = %q, want 6 chars", r.Code)
	}
	if r.Members[0].UserID != "u1" || !r.Members[0].IsHost {
		t.Fatal("host not seated at seat 0")
	}
	if !r.Members[0].Ready {
		t.Fatal("room creator must be ready by default")
	}
	got, err := m.ByCode(r.Code)
	if err != nil || got.ID != r.ID {
		t.Fatalf("ByCode: %v %v", got.ID, err)
	}
}

func TestJoinSeatAllocation(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	for i := 2; i <= 4; i++ {
		r, err := m.Join(r.Code, string(rune('0'+i)), "P", "", "")
		if err != nil {
			t.Fatalf("join %d: %v", i, err)
		}
		_ = r
	}
	final, _ := m.Get(r.ID)
	if len(final.Members) != 4 {
		t.Fatalf("members = %d, want 4", len(final.Members))
	}
	seats := map[int]bool{}
	for _, mem := range final.Members {
		if seats[mem.Seat] {
			t.Fatalf("duplicate seat %d", mem.Seat)
		}
		seats[mem.Seat] = true
	}
	// Fifth join must fail.
	if _, err := m.Join(r.Code, "u9", "Late", "", ""); !errors.Is(err, ErrRoomFull) {
		t.Fatalf("want ErrRoomFull, got %v", err)
	}
	// Rejoin rejected.
	if _, err := m.Join(r.Code, "u1", "Alice", "", ""); !errors.Is(err, ErrAlreadyInRoom) {
		t.Fatalf("want ErrAlreadyInRoom, got %v", err)
	}
}

func TestLeaveTransfersHostAndDeletesEmpty(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")
	_, err := m.Leave(r.ID, "u1")
	if err != nil {
		t.Fatalf("Leave: %v", err)
	}
	got, _ := m.Get(r.ID)
	if got.HostID != "u2" {
		t.Fatalf("host = %q, want u2", got.HostID)
	}
	if got.Members[0].IsHost != true {
		t.Fatal("new host flag not set")
	}
	// Last member leaves → room deleted.
	_, _ = m.Leave(r.ID, "u2")
	if _, err := m.Get(r.ID); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("empty room should be deleted, got %v", err)
	}
}

func TestReadyAndStart(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")
	if _, err := m.SetReady(r.ID, "u2", true); err != nil {
		t.Fatalf("SetReady: %v", err)
	}
	got, _ := m.Get(r.ID)
	if !got.Members[1].Ready {
		t.Fatal("ready flag not set")
	}
	if _, err := m.SetReady(r.ID, "ghost", true); !errors.Is(err, ErrNotInRoom) {
		t.Fatalf("want ErrNotInRoom, got %v", err)
	}
}

func TestKickRules(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")
	_, _ = m.Join(r.Code, "u3", "Cara", "", "")

	if _, err := m.Kick(r.ID, "u2", "u3"); !errors.Is(err, ErrNotHost) {
		t.Fatalf("non-host kick: %v", err)
	}
	if _, err := m.Kick(r.ID, "u1", "u1"); !errors.Is(err, ErrCannotKickSelf) {
		t.Fatalf("self kick: %v", err)
	}
	got, err := m.Kick(r.ID, "u1", "u3")
	if err != nil {
		t.Fatalf("kick: %v", err)
	}
	if got.InSeat("u3") {
		t.Fatal("kicked member still seated")
	}
}

func TestAISlots(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Private, defaultSettings())
	got, err := m.AddAI(r.ID, "u1", "hard", "Robot")
	if err != nil {
		t.Fatalf("AddAI: %v", err)
	}
	var ai *Member
	for i := range got.Members {
		if got.Members[i].IsAI {
			ai = &got.Members[i]
		}
	}
	if ai == nil || !ai.Ready || ai.AIDifficulty != "hard" {
		t.Fatalf("AI member wrong: %+v", ai)
	}
	// Non-host cannot add.
	if _, err := m.AddAI(r.ID, "ghost", "easy", "Bot2"); !errors.Is(err, ErrNotHost) {
		t.Fatalf("want ErrNotHost, got %v", err)
	}
	// Remove AI.
	got, err = m.RemoveAI(r.ID, "u1", ai.UserID)
	if err != nil {
		t.Fatalf("RemoveAI: %v", err)
	}
	for _, mem := range got.Members {
		if mem.IsAI {
			t.Fatal("AI still present")
		}
	}
}

func TestFillEmptyWithAI(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	got, err := m.FillEmptyWithAI(r.ID, "u1")
	if err != nil {
		t.Fatalf("FillEmptyWithAI: %v", err)
	}
	if len(got.Members) != 4 {
		t.Fatalf("members = %d, want 4", len(got.Members))
	}
	aiCount := 0
	for _, mem := range got.Members {
		if mem.IsAI {
			aiCount++
			if !mem.Ready {
				t.Fatal("AI must be ready")
			}
			if mem.AIDifficulty == "" {
				t.Fatal("AI difficulty empty")
			}
		}
	}
	if aiCount != 3 {
		t.Fatalf("AI count = %d, want 3", aiCount)
	}
	if _, err := m.FillEmptyWithAI(r.ID, "u1"); !errors.Is(err, ErrNoEmptySlot) {
		t.Fatalf("want ErrNoEmptySlot, got %v", err)
	}
	if _, err := m.FillEmptyWithAI(r.ID, "ghost"); !errors.Is(err, ErrNotHost) {
		t.Fatalf("want ErrNotHost, got %v", err)
	}
}

func TestListPublicOnly(t *testing.T) {
	m := newTestManager()
	pub, _ := m.Create("u1", "A", "", "", "Public Room", Public, defaultSettings())
	_, _ = m.Create("u2", "B", "", "", "Private Room", Private, defaultSettings())
	list := m.ListPublic()
	if len(list) != 1 || list[0].ID != pub.ID {
		t.Fatalf("public list = %+v", list)
	}
}

func TestNotifierFires(t *testing.T) {
	m := newTestManager()
	updates := make(chan Room, 8)
	m.SetNotifier(chanNotifier(updates))
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")
	select {
	case got := <-updates:
		if got.HostID != "u1" {
			t.Fatalf("unexpected snapshot host %q", got.HostID)
		}
	default:
		t.Fatal("no notification received")
	}
}

type chanNotifier chan Room

func (c chanNotifier) RoomUpdated(r Room) { c <- r }

func TestCodesUniqueAcrossManyRooms(t *testing.T) {
	m := newTestManager()
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		r, err := m.Create("u", "A", "", "", "Room", Public, defaultSettings())
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		if seen[r.Code] {
			t.Fatalf("duplicate code %q", r.Code)
		}
		seen[r.Code] = true
	}
}

func seatOf(r Room, userID string) int {
	for _, m := range r.Members {
		if m.UserID == userID {
			return m.Seat
		}
	}
	return -1
}

func TestMoveSeatSwapAndEmpty(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")

	got, err := m.MoveSeat(r.ID, "u1", 0, 2)
	if err != nil {
		t.Fatalf("move to empty: %v", err)
	}
	if seatOf(got, "u1") != 2 {
		t.Fatalf("alice seat = %d, want 2", seatOf(got, "u1"))
	}
	if seatOf(got, "u2") != 1 {
		t.Fatalf("bob seat = %d, want 1", seatOf(got, "u2"))
	}

	got, err = m.MoveSeat(r.ID, "u1", 2, 1)
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	if seatOf(got, "u1") != 1 || seatOf(got, "u2") != 2 {
		t.Fatalf("after swap alice=%d bob=%d, want 1 and 2", seatOf(got, "u1"), seatOf(got, "u2"))
	}
}

func TestMoveSeatRejected(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")

	if _, err := m.MoveSeat(r.ID, "u2", 1, 2); !errors.Is(err, ErrNotHost) {
		t.Fatalf("non-host: %v", err)
	}
	if _, err := m.MoveSeat(r.ID, "u1", -1, 2); !errors.Is(err, ErrInvalidSeat) {
		t.Fatalf("invalid seat: %v", err)
	}
	if _, err := m.MoveSeat(r.ID, "u1", 3, 2); !errors.Is(err, ErrEmptySeat) {
		t.Fatalf("empty source: %v", err)
	}

	if err := m.MarkStarted(r.ID); err != nil {
		t.Fatalf("MarkStarted: %v", err)
	}
	if _, err := m.MoveSeat(r.ID, "u1", 0, 2); !errors.Is(err, ErrGameInProgress) {
		t.Fatalf("in-game move: %v", err)
	}
}

func TestDeleteLobbyOnly(t *testing.T) {
	m := newTestManager()
	r, _ := m.Create("u1", "Alice", "", "", "Room", Public, defaultSettings())
	_, _ = m.Join(r.Code, "u2", "Bob", "", "")

	if err := m.Delete(r.ID, "u2"); !errors.Is(err, ErrNotHost) {
		t.Fatalf("non-host delete: %v", err)
	}
	if err := m.Delete(r.ID, "u1"); err != nil {
		t.Fatalf("delete lobby: %v", err)
	}
	if _, err := m.Get(r.ID); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("deleted room still present: %v", err)
	}

	r2, _ := m.Create("u1", "Alice", "", "", "Match", Public, defaultSettings())
	if err := m.MarkStarted(r2.ID); err != nil {
		t.Fatalf("MarkStarted: %v", err)
	}
	if err := m.Delete(r2.ID, "u1"); !errors.Is(err, ErrGameInProgress) {
		t.Fatalf("in-game delete: %v", err)
	}
}
