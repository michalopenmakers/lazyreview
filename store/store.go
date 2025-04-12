package store

import (
	"sync"
)

type DataStore struct {
	Data map[string]string
}

var storeMutex = &sync.Mutex{}
var instance *DataStore

func GetStore() *DataStore {
	if instance == nil {
		storeMutex.Lock()
		defer storeMutex.Unlock()
		if instance == nil {
			instance = &DataStore{Data: make(map[string]string)}
		}
	}
	return instance
}
