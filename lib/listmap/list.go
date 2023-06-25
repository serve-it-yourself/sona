package listmap

import (
	"log"
	"sync"
	"sync/atomic"
)

type ListMap struct {
	mapping map[string]int
	list    []atomic.Pointer[func([]byte)]
	lock    sync.RWMutex
}

func New() *ListMap {
	return &ListMap{
		mapping: make(map[string]int),
		list:    make([]atomic.Pointer[func([]byte)], 0),
	}
}

func (l *ListMap) Append(key string, value func([]byte)) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if index, ok := l.mapping[key]; ok {
		l.list[index].Store(&value)
	} else {
		for i := 0; i < len(l.list); i++ {
			if l.list[i].CompareAndSwap(nil, &value) {
				return
			}
		}
		l.mapping[key] = len(l.list)
		l.list = append(l.list, atomic.Pointer[func([]byte)]{})
		l.list[len(l.list)-1].Store(&value)
	}
}

func (l *ListMap) Remove(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if index, ok := l.mapping[key]; ok {
		l.list[index].Store(nil)
		delete(l.mapping, key)
	}
}

func (l *ListMap) Iterate(data []byte) {
	for i := 0; i < len(l.list); i++ {
		if value := l.list[i].Load(); value != nil {
			func() {
				defer func() {
					if err := recover(); err != nil {
						log.Println(err)
					}
				}()
				(*value)(data)
			}()
		}
	}
}
