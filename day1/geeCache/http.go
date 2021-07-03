package geeCache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

const (
	DefaultBasePath = "/http"
	defaultReplicas = 50
)

type HttpPool struct {
	self        string
	basePath    string
	mu          sync.Mutex // guards peers and httpGetters
	peers       *ConsistentHashMap
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

func NewHttpPool(self string) *HttpPool {
	return &HttpPool{
		self:     self,
		basePath: DefaultBasePath,
	}
}

func (p *HttpPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = NewConsistentHash(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

func (p *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

func (p *HttpPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", format, v)
}

func (p *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		w.Write([]byte("HTTP Pool serving unexpected path: " + r.URL.Path))
		//panic("HTTPPool serving unexpected path: " + r.URL.Path)
		return
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	parts := strings.Split(r.URL.Path[len(p.basePath)+1:], "/")
	fmt.Println("parts", parts)
	if len(parts) > 2 {
		http.Error(w, "路径不合法", http.StatusInternalServerError)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
