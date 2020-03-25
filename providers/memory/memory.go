package memory

import (
	"container/list"
	"fmt"
	"github.com/woguolufei/session"
	"sync"
	"time"
)

var p *Provider

func init() {
	p = &Provider{
		list:     list.New(),
		sessions: make(map[string]*list.Element, 0),
	}

	session.Register("memory", p)
}

type Session struct {
	sid          string
	timeAccessed time.Time
	value        map[interface{}]interface{}
}

func (s Session) Set(key, value interface{}) error {
	s.value[key] = value
	p.Update(s.sid)
	return nil
}

func (s Session) Get(key interface{}) interface{} {
	p.Update(s.sid)
	if v, ok := s.value[key]; ok {
		return v
	}
	return nil
}

func (s Session) Delete(key interface{}) error {
	delete(s.value, key)
	p.Update(s.sid)
	return nil
}

func (s Session) SessionId() string {
	return s.sid
}

type Provider struct {
	lock     sync.Mutex
	sessions map[string]*list.Element
	list     *list.List
}

func (p Provider) Init(sid string) (session.Session, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	sess := &Session{
		sid:          sid,
		timeAccessed: time.Now(),
		value:        v,
	}
	p.sessions[sid] = p.list.PushFront(sess)
	return sess, nil
}

func (p Provider) Read(sid string) (session.Session, error) {
	if element, ok := p.sessions[sid]; ok {
		return element.Value.(*Session), nil
	} else {
		sess, err := p.Init(sid)
		return sess, err
	}
}

func (p Provider) Destroy(sid string) error {
	if element, ok := p.sessions[sid]; ok {
		delete(p.sessions, sid)
		p.list.Remove(element)
		return nil
	}
	return nil
}

func (p Provider) GC(maxLifeTime int64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for {
		element := p.list.Back()
		if element == nil {
			break
		}
		if element.Value.(*Session).timeAccessed.Unix()+maxLifeTime < time.Now().Unix() {
			p.list.Remove(element)
			delete(p.sessions, element.Value.(*Session).sid)
		} else {
			break
		}
	}
}

func (p Provider) Update(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	fmt.Println("hehe")
	if element, ok := p.sessions[sid]; ok {
		element.Value.(*Session).timeAccessed = time.Now()
		p.list.MoveToFront(element)
	}
	return nil
}
