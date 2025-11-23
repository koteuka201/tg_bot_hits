package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type Storage struct {
	data map[int64]map[string]string
	mu   sync.RWMutex
	file string
}

func NewStorage(file string) (*Storage, error) {
	s := &Storage{
		data: make(map[int64]map[string]string),
		file: file,
	}

	if _, err := os.Stat(file); err == nil {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		if len(b) > 0 {
			if err := json.Unmarshal(b, &s.data); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

// saveUnlocked сохраняет данные БЕЗ захвата мьютекса
// Должна вызываться только внутри функций, которые уже владеют mu
func (s *Storage) saveUnlocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.file, b, 0644)
}

func (s *Storage) GetUserField(userID int64, key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.data[userID]; ok {
		return user[key]
	}
	return ""
}

func (s *Storage) SetUserField(userID int64, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[userID]; !ok {
		s.data[userID] = make(map[string]string)
	}
	s.data[userID][key] = value
	if err := s.saveUnlocked(); err != nil {
		log.Printf("failed to save storage: %v", err)
	}
}

func (s *Storage) HasUserField(userID int64, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.data[userID]; ok {
		_, exists := user[key]
		return exists
	}
	return false
}

func (s *Storage) DeleteUserField(userID int64, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.data[userID]; ok {
		delete(user, key)
		if err := s.saveUnlocked(); err != nil {
			log.Printf("failed to save storage: %v", err)
		}
	}
}

func (s *Storage) GetUserMap(userID int64) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if user, ok := s.data[userID]; ok {
		// возвращаем копию, чтобы внешние изменения не портили оригинал
		cpy := make(map[string]string)
		for k, v := range user {
			cpy[k] = v
		}
		return cpy
	}
	return make(map[string]string)
}