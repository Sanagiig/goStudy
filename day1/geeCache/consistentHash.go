package geeCache

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type ConsistentHashMap struct {
	hash     Hash
	replicas int
	keys     []int
	hashMap  map[int]string
}

func NewConsistentHash(replicas int, hash Hash) *ConsistentHashMap {
	m := &ConsistentHashMap{
		replicas: replicas,
		hash:     hash,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *ConsistentHashMap) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

func (m *ConsistentHashMap) Get(key string) string {
	klen := len(m.keys)
	if klen == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(klen, func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%klen]]
}
