package room

import (
	"github.com/google/uuid"
	"mrs/internal/dto"
	"sync"
	"time"
)

type Room struct {
	mu        sync.RWMutex
	id        uuid.UUID    // айди комнаты
	queue     []*dto.Video // очередь видео
	current   *dto.Video   // текущее видео
	playing   bool         // флаг того, что играет
	basePos   float64      // время видео для синхронизации
	startedAt time.Time    // время начала действия какого то

	subscribers map[int]chan dto.State // пользователи
	nextSubID   int                    // айди для пользоввателй

	createAt time.Time
}

// слепок который отдаем пользователям он к ним привязан

func (r *Room) broadcastLocked(state dto.State) {
	for userID, ch := range r.subscribers {
		select {
		case ch <- state:
		default:
			delete(r.subscribers, userID)
			close(ch)
		}
	}
}

func (r *Room) stateLock(now time.Time) dto.State {
	pos := r.basePos
	if r.playing {
		pos += now.Sub(r.startedAt).Seconds()
	}

	q := make([]*dto.Video, len(r.queue))
	copy(q, r.queue)

	return dto.State{
		ID:        r.id,
		Current:   r.current,
		Queue:     q,
		Playing:   r.playing,
		Position:  pos,
		UpdatedAt: now,
	}
}
