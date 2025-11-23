package room

import (
	"fmt"
	"github.com/google/uuid"
	"mrs/internal/dto"
	"sync"
	"time"
)

type ServiceRoom struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]*Room
}

func NewServiceRoom(interval, idle time.Duration) *ServiceRoom {
	serviceRoom := &ServiceRoom{rooms: make(map[uuid.UUID]*Room)}
	go serviceRoom.StartCleanupWorker(interval, idle)

	return serviceRoom
}

func (rs *ServiceRoom) CreateRoom() (uuid.UUID, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	id := uuid.New()
	rs.rooms[id] = &Room{
		id:          id,
		queue:       make([]*dto.Video, 0, 100),
		subscribers: make(map[int]chan dto.State),
		nextSubID:   1,
		createAt:    time.Now(),
	}
	return id, nil
}

func (rs *ServiceRoom) RemoveRoom(id uuid.UUID) {
	// 1. Забираем комнату и удаляем из map под одним локом
	rs.mu.Lock()
	room, ok := rs.rooms[id]
	if !ok {
		rs.mu.Unlock()
		return
	}
	delete(rs.rooms, id)
	rs.mu.Unlock()

	// 2. Под локом самой комнаты закрываем всех подписчиков
	room.mu.Lock()
	for userID, ch := range room.subscribers {
		delete(room.subscribers, userID)
		close(ch)
	}
	room.mu.Unlock()
}

func (rs *ServiceRoom) ConnectToTheRoom(id uuid.UUID) (int, <-chan dto.State, error) {
	room, err := rs.getRoom(id)
	if err != nil {
		return -1, nil, err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	userID := room.nextSubID
	room.nextSubID++

	ch := make(chan dto.State, 10)
	room.subscribers[userID] = ch

	// считаем актуальное состояние
	now := time.Now()

	st := room.stateLock(now)

	// первая отправка
	ch <- st

	return userID, ch, nil
}

func (rs *ServiceRoom) DisconnectUser(id uuid.UUID, userID int) error {
	// ищем комнату по id
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	// комнату нашли удаляем юзера
	ch, ok := room.subscribers[userID]
	if !ok {
		return fmt.Errorf("user does not exist")
	}

	delete(room.subscribers, userID)
	close(ch)

	if len(room.subscribers) == 0 {
		go rs.RemoveRoom(id)
	}

	return nil
}

func (rs *ServiceRoom) Play(id uuid.UUID) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.playing {
		return nil
	}

	if room.current == nil {
		if len(room.queue) == 0 {
			return fmt.Errorf("queue is empty")
		}
		room.basePos = 0
		room.current = room.queue[0]
		room.queue = room.queue[1:]
	}

	now := time.Now()
	room.startedAt = now
	room.playing = true

	st := room.stateLock(now)
	room.broadcastLocked(st)
	return nil
}

func (rs *ServiceRoom) Pause(id uuid.UUID) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if !room.playing {
		return nil
	}

	if room.current == nil {
		return nil
	}

	now := time.Now()
	pos := room.basePos + now.Sub(room.startedAt).Seconds()

	room.basePos = pos
	room.startedAt = now
	room.playing = false

	st := room.stateLock(now)
	room.broadcastLocked(st)

	return nil
}

func (rs *ServiceRoom) Next(id uuid.UUID) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if len(room.queue) == 0 {
		now := time.Now()

		room.basePos = 0
		room.current = nil
		room.startedAt = now
		room.playing = false

		st := room.stateLock(now)
		room.broadcastLocked(st)

		return nil
	}

	now := time.Now()

	room.basePos = 0
	room.current = room.queue[0]
	room.queue = room.queue[1:]
	room.startedAt = now
	room.playing = true

	st := room.stateLock(now)
	room.broadcastLocked(st)

	return nil
}

func (rs *ServiceRoom) AddVideoInQueue(id uuid.UUID, video *dto.Video) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()
	room.queue = append(room.queue, video)

	now := time.Now()

	st := room.stateLock(now)
	room.broadcastLocked(st)
	return nil
}

func (rs *ServiceRoom) getRoom(id uuid.UUID) (*Room, error) {
	rs.mu.RLock()
	room, ok := rs.rooms[id]
	rs.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("room does not exist")
	}

	return room, nil
}

func (rs *ServiceRoom) GetAllRoomsInfo() []*dto.Room {
	// 1. Делаем срез комнат под локом сервиса
	rs.mu.RLock()
	rooms := make([]*Room, 0, len(rs.rooms))
	for _, room := range rs.rooms {
		rooms = append(rooms, room)
	}
	rs.mu.RUnlock()

	results := make([]*dto.Room, 0, len(rooms))

	// 2. Для каждой комнаты берём её лок и копируем нужные поля
	for _, room := range rooms {
		room.mu.RLock()

		// копия очереди, чтобы не светить внутренний слайс наружу
		queueCopy := make([]*dto.Video, len(room.queue))
		copy(queueCopy, room.queue)

		res := &dto.Room{
			ID:          room.id,
			Queue:       queueCopy,
			Current:     room.current, // Video считаем иммутабельной
			Playing:     room.playing,
			Subscribers: len(room.subscribers),
		}
		room.mu.RUnlock()

		results = append(results, res)
	}

	return results
}

func (rs *ServiceRoom) DeleteVideoInQueue(id uuid.UUID, idx int) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if idx < 0 || idx >= len(room.queue) {
		return fmt.Errorf("index out of range")
	}

	room.queue = append(room.queue[:idx], room.queue[idx+1:]...)

	now := time.Now()
	st := room.stateLock(now)
	room.broadcastLocked(st)

	return nil
}

func (rs *ServiceRoom) Seek(id uuid.UUID, pos float64) error {
	room, err := rs.getRoom(id)
	if err != nil {
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.current == nil {
		return fmt.Errorf("no current track")
	}

	// Нормализуем позицию
	if pos < 0 {
		pos = 0
	}
	if pos > float64(room.current.Duration) {
		pos = float64(room.current.Duration)
	}

	now := time.Now()

	// КЛЮЧЕВОЕ: здесь мы «фиксируем перемотку»
	room.basePos = pos

	// если трек сейчас играет — считаем, что он играет с новой точки
	if room.playing {
		room.startedAt = now
	}

	// собираем новый state обычным способом
	st := room.stateLock(now)
	room.broadcastLocked(st)

	return nil
}

func (rs *ServiceRoom) StartCleanupWorker(interval, idle time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rs.cleanupRooms(idle)
		}
	}
}

func (rs *ServiceRoom) cleanupRooms(idle time.Duration) {
	now := time.Now()

	rs.mu.RLock()

	var ids []uuid.UUID
	for id, room := range rs.rooms {
		room.mu.RLock()
		empty := len(room.subscribers) == 0
		idleTooLong := now.Sub(room.createAt) > idle
		room.mu.RUnlock()

		if empty && idleTooLong {
			ids = append(ids, id)
		}
	}
	rs.mu.RUnlock()

	for _, id := range ids {
		rs.RemoveRoom(id)
	}
}
