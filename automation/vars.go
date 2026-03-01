package automation

import (
	"maps"
	"sync"
)

type VarStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewVarStore() *VarStore {
	return &VarStore{
		data: make(map[string]string),
	}
}

func (v *VarStore) Set(key, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data[key] = value
}

func (v *VarStore) Get(key string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	val, ok := v.data[key]
	return val, ok
}

func (v *VarStore) All() map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	copy := make(map[string]string)
	maps.Copy(copy, v.data)
	return copy
}
