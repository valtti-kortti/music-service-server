package ws_transport

import (
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"log"
	"mrs/internal/dto"
	http_transport "mrs/internal/transport/http"
	"net/http"
)

var (
	pause = "pause"
	play  = "play"
	next  = "next"
)

type ServiceRoom interface {
	ConnectToTheRoom(id uuid.UUID) (int, <-chan dto.State, error)
	DisconnectUser(id uuid.UUID, userID int) error
	AddVideoInQueue(id uuid.UUID, video *dto.Video) error

	Play(id uuid.UUID) error
	Pause(id uuid.UUID) error
	Next(id uuid.UUID) error
}

type WSHandler struct {
	service ServiceRoom
}

func NewWSHandler(service ServiceRoom) *WSHandler {
	return &WSHandler{service: service}
}

func (h *WSHandler) RoomWS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http_transport.WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http_transport.WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	userID, chState, err := h.service.ConnectToTheRoom(id)
	if err != nil {
		log.Println(err)
		h.service.DisconnectUser(id, userID)
		http_transport.WriteJsonError(w, http.StatusBadRequest, "room_id is invalid")
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	defer conn.Close(websocket.StatusNormalClosure, "bye")
	defer h.service.DisconnectUser(id, userID)

	go func() {
		for {
			select {
			case state, ok := <-chState:
				if !ok {
					return
				}
				if err = wsjson.Write(ctx, conn, state); err != nil {
					log.Println(err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		var cmd dto.Command
		if err = wsjson.Read(ctx, conn, &cmd); err != nil {
			log.Println(err)
			return
		}

		switch cmd.Type {
		case play:
			_ = h.service.Play(id)
		case pause:
			_ = h.service.Pause(id)
		case next:
			_ = h.service.Next(id)
		}

	}

}
