package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hokm/platform/internal/room"
)

func (s *Server) roomErr(err error) error {
	switch {
	case errors.Is(err, room.ErrRoomNotFound):
		return apiError(http.StatusNotFound, "room_not_found", "room not found")
	case errors.Is(err, room.ErrRoomFull):
		return apiError(http.StatusConflict, "room_full", "room is full")
	case errors.Is(err, room.ErrAlreadyInRoom):
		return apiError(http.StatusConflict, "already_in_room", "already in this room")
	case errors.Is(err, room.ErrNotInRoom):
		return apiError(http.StatusNotFound, "not_in_room", "not a member of this room")
	case errors.Is(err, room.ErrNotHost):
		return apiError(http.StatusForbidden, "not_host", "only the host can do that")
	case errors.Is(err, room.ErrGameInProgress):
		return apiError(http.StatusConflict, "game_in_progress", "game already started")
	case errors.Is(err, room.ErrInvalidName):
		return apiError(http.StatusUnprocessableEntity, "validation", "room name must be 2-40 characters")
	case errors.Is(err, room.ErrInvalidVisibility):
		return apiError(http.StatusUnprocessableEntity, "validation", "visibility must be public or private")
	case errors.Is(err, room.ErrInvalidRoundCount):
		return apiError(http.StatusUnprocessableEntity, "validation", "invalid round count")
	case errors.Is(err, room.ErrInvalidGameSpeed):
		return apiError(http.StatusUnprocessableEntity, "validation", "game speed must be fast, medium or slow")
	case errors.Is(err, room.ErrCannotKickSelf):
		return apiError(http.StatusUnprocessableEntity, "validation", "cannot kick yourself")
	case errors.Is(err, room.ErrNoEmptySlot):
		return apiError(http.StatusConflict, "no_empty_slot", "no empty seat")
	case errors.Is(err, room.ErrNotAnAI):
		return apiError(http.StatusUnprocessableEntity, "not_ai", "member is not an AI")
	default:
		return err
	}
}

func (s *Server) roomFromRequest(w http.ResponseWriter, r *http.Request, idParam string) (*room.Room, string, bool) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return nil, "", false
	}
	roomID := r.PathValue(idParam)
	rm, err := s.rooms.Get(roomID)
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return nil, uid, false
	}
	return &rm, uid, true
}

type createRoomRequest struct {
	Name        string `json:"name"`
	Visibility  string `json:"visibility"`
	RoundCount  int    `json:"round_count"`
	GameSpeed   string `json:"game_speed"`
	ChatEnabled *bool  `json:"chat_enabled"`
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return
	}
	var req createRoomRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	// Sensible defaults when the creator omits fields.
	if req.RoundCount == 0 {
		req.RoundCount = 1
	}
	if req.GameSpeed == "" {
		req.GameSpeed = "medium"
	}
	chatEnabled := true
	if req.ChatEnabled != nil {
		chatEnabled = *req.ChatEnabled
	}
	settings := room.RoomSettings{
		RoundCount:  req.RoundCount,
		GameSpeed:   req.GameSpeed,
		ChatEnabled: chatEnabled,
	}
	username := s.usernameFor(uid)
	rm, err := s.rooms.Create(uid, username, req.Name, room.Visibility(req.Visibility), settings)
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"room": rm})
}

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"rooms": s.rooms.ListPublic()})
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	rm, _, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": rm})
}

type joinRoomRequest struct {
	Code string `json:"code"`
}

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return
	}
	var req joinRoomRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	rm, err := s.rooms.Join(req.Code, uid, s.usernameFor(uid))
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": rm})
}

func (s *Server) handleLeaveRoom(w http.ResponseWriter, r *http.Request) {
	rm, uid, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.rooms.Leave(rm.ID, uid); err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

type readyRequest struct {
	Ready bool `json:"ready"`
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	rm, uid, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	var req readyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	out, err := s.rooms.SetReady(rm.ID, uid, req.Ready)
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": out})
}

type kickRequest struct {
	UserID string `json:"user_id"`
}

func (s *Server) handleKick(w http.ResponseWriter, r *http.Request) {
	rm, uid, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	var req kickRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	out, err := s.rooms.Kick(rm.ID, uid, req.UserID)
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": out})
}

type addAIRequest struct {
	Difficulty string `json:"difficulty"`
}

func (s *Server) handleAddAI(w http.ResponseWriter, r *http.Request) {
	rm, uid, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	var req addAIRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	switch req.Difficulty {
	case "easy", "medium", "hard", "expert", "pro":
	default:
		writeError(w, r, apiError(http.StatusUnprocessableEntity, "validation",
			"difficulty must be easy, medium, hard, expert or pro"))
		return
	}
	out, err := s.rooms.AddAI(rm.ID, uid, req.Difficulty, "AI ("+req.Difficulty+")")
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": out})
}

type removeAIRequest struct {
	UserID string `json:"user_id"`
}

func (s *Server) handleRemoveAI(w http.ResponseWriter, r *http.Request) {
	rm, uid, ok := s.roomFromRequest(w, r, "id")
	if !ok {
		return
	}
	var req removeAIRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, r, apiError(http.StatusBadRequest, "bad_request", "invalid JSON body"))
		return
	}
	out, err := s.rooms.RemoveAI(rm.ID, uid, req.UserID)
	if err != nil {
		writeError(w, r, s.roomErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"room": out})
}
