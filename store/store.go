package store

import (
	"errors"
	"sync"
)

var (
	hashStorage = make(map[string]string)
	mu          sync.Mutex
)

func GetLastHash(reviewID string) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	if h, ok := hashStorage[reviewID]; ok {
		return h, nil
	}
	return "", errors.New("hash not found")
}

func UpdateLastHash(reviewID string, newHash string) error {
	mu.Lock()
	defer mu.Unlock()
	hashStorage[reviewID] = newHash
	return nil
}
