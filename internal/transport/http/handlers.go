package http_transport

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"mrs/internal/dto"
	"net/http"
	"strconv"
)

type ServiceYoutube interface {
	GetListVideo(ctx context.Context, query string) ([]*dto.Video, error)
}

type ServiceRoom interface {
	CreateRoom() (uuid.UUID, error)
	AddVideoInQueue(id uuid.UUID, video *dto.Video) error
	GetAllRoomsInfo() []*dto.Room
	DeleteVideoInQueue(id uuid.UUID, idx int) error
	Seek(id uuid.UUID, pos float64) error
}

type Handler struct {
	servYoutube ServiceYoutube
	servRoom    ServiceRoom
}

func NewHandler(service ServiceYoutube, servRoom ServiceRoom) *Handler {
	return &Handler{servYoutube: service, servRoom: servRoom}
}

func (h *Handler) GetListVideo(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("name")
	if query == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter name is required")
		return
	}

	res, err := h.servYoutube.GetListVideo(r.Context(), query)
	if err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	id, err := h.servRoom.CreateRoom()
	if err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	resp := dto.ResponseRoom{ID: id}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *Handler) AddVideoInQueue(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	var video dto.Video

	if err := json.NewDecoder(r.Body).Decode(&video); err != nil {
		WriteJsonError(w, http.StatusBadRequest, "body is required")
		return
	}

	err = h.servRoom.AddVideoInQueue(id, &video)
	if err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)

}

func (h *Handler) GetAllRoomsInfo(w http.ResponseWriter, r *http.Request) {
	rooms := h.servRoom.GetAllRoomsInfo()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *Handler) DeleteVideoInQueue(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	idxStr := r.URL.Query().Get("idx")
	if idxStr == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter idx is required")
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		WriteJsonError(w, http.StatusBadRequest, "query parameter idx is required")
	}

	err = h.servRoom.DeleteVideoInQueue(id, idx)
	if err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Seek(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		WriteJsonError(w, http.StatusBadRequest, "query parameter id is required")
		return
	}

	posStr := r.URL.Query().Get("pos")
	if posStr == "" {
		WriteJsonError(w, http.StatusBadRequest, "query parameter pos is required")
	}

	pos, err := strconv.ParseFloat(posStr, 64)
	if err != nil {
		WriteJsonError(w, http.StatusBadRequest, "query parameter pos is required")
	}

	if err := h.servRoom.Seek(id, pos); err != nil {
		WriteJsonError(w, http.StatusInternalServerError, err.Error())
	}

	w.WriteHeader(http.StatusOK)
}
