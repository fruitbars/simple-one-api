package config

import (
	"hash/fnv"
	"math/rand"
	"simple-one-api/pkg/mycomdef"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	newIdx := atomic.AddUint32(idx, 1)
	return int(newIdx) % n
}

func getHashIndex(key string, n int) int {
	// 包含到毫秒的时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05.999")
	h := fnv.New32a()
	h.Write([]byte(key + timestamp))
	return int(h.Sum32()) % n
}

func GetLBIndex(lbStrategy string, key string, length int) int {
	lbs := strings.ToLower(lbStrategy)
	switch lbs {
	case mycomdef.KEYNAME_FIRST:
		return 0
	case mycomdef.KEYNAME_RANDOM, mycomdef.KEYNAME_RAND:
		return getRandomIndex(length)
	case mycomdef.KEYNAME_ROUND_ROBIN, mycomdef.KEYNAME_RR:
		return getRoundRobinIndex(key, length)
	case mycomdef.KEYNAME_HASH:
		return getHashIndex(key, length)
	default:
		return getRandomIndex(length)
	}
}
