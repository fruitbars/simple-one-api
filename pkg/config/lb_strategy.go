package config

import (
	"hash/fnv"
	"math/rand"
	"simple-one-api/pkg/mycomdef"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	rrIndices = make(map[string]*uint32)
	randLock  = &sync.Mutex{}
	modelLock = &sync.RWMutex{}
)

func getRandomIndex(n int) int {
	randLock.Lock()
	defer randLock.Unlock()
	return rand.Intn(n)
}

func getRoundRobinIndex(modelName string, n int) int {
	modelLock.RLock()
	idx, exists := rrIndices[modelName]
	modelLock.RUnlock()

	if !exists {
		modelLock.Lock()
		if idx, exists = rrIndices[modelName]; !exists { // double check locking
			var newIndex uint32 = 0
			rrIndices[modelName] = &newIndex
			idx = &newIndex
		}
		modelLock.Unlock()
	}

	// Increment index atomically and get the server
	newIdx := atomic.AddUint32(idx, 1) - 1
	return int(newIdx) % n
}

func getHashIndex(key string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(n))
}

func GetLBIndex(lbStrategy string, key string, length int) int {
	if length <= 0 {
		return 0
	}
	lbs := strings.ToLower(strings.TrimSpace(lbStrategy))
	switch lbs {
	case mycomdef.KEYNAME_FIRST:
		return 0
	case mycomdef.KEYNAME_RANDOM, mycomdef.KEYNAME_RAND:
		return getRandomIndex(length)
	case mycomdef.KEYNAME_ROUND_ROBIN, "round_robin", mycomdef.KEYNAME_RR:
		return getRoundRobinIndex(key, length)
	case mycomdef.KEYNAME_HASH:
		return getHashIndex(key, length)
	default:
		return getRandomIndex(length)
	}
}
