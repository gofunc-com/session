package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Session interface {
	Set(key, value interface{}) error
	Get(key interface{}) interface{}
	Delete(key interface{}) error
	SessionId() string
}

type Provider interface {
	Init(sid string) (Session, error)
	Read(sid string) (Session, error)
	Destroy(sid string) error
	GC(maxLifeTime int64)
}

var providers = make(map[string]Provider)

func Register(name string, provider Provider) {
	if provider == nil {
		panic("session 服务提供者不能为nil")
	}
	if _, dup := providers[name]; dup {
		panic("session 服务提供者不能注册两次")
	}
	providers[name] = provider
}

type Manager struct {
	cookieName  string
	lock        sync.Mutex
	provider    Provider
	maxLiefTime int64
}

func NewManager(providerName, cookieName string, maxLifeTime int64) (*Manager, error) {
	provider, ok := providers[providerName]
	if !ok {
		return nil, fmt.Errorf("session: %v 服务未注册", providerName)
	}

	return &Manager{cookieName: cookieName, provider: provider, maxLiefTime: maxLifeTime}, nil
}

func (manager *Manager) Id() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (manager *Manager) Start(w http.ResponseWriter, r *http.Request) (session Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		sid := manager.Id()
		session, _ = manager.provider.Init(sid)
		http.SetCookie(w, &http.Cookie{
			Name:     manager.cookieName,
			Value:    url.QueryEscape(sid),
			Path:     "/",
			HttpOnly: true,
			MaxAge:   int(manager.maxLiefTime),
		})
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = manager.provider.Read(sid)
	}

	fmt.Println(manager.provider)
	return
}

func (manager *Manager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		return
	}
	manager.lock.Lock()
	defer manager.lock.Unlock()

	sid, _ := url.QueryUnescape(cookie.Value)
	manager.provider.Destroy(sid)
	http.SetCookie(w, &http.Cookie{
		Name:     manager.cookieName,
		Path:     "/",
		Expires:  time.Now(),
		MaxAge:   -1,
		HttpOnly: true,
	})
}

func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.provider.GC(manager.maxLiefTime)
}
