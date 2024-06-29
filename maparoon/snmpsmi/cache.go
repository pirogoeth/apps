package snmpsmi

import (
	"sync"

	"github.com/sleepinggenius2/gosmi"
)

type oidCache struct {
	cacheMap map[string]*gosmi.SmiNode
	lock     *sync.Mutex
}

func newOidCache() *oidCache {
	return &oidCache{
		cacheMap: make(map[string]*gosmi.SmiNode),
		lock:     new(sync.Mutex),
	}
}

func (c *oidCache) Get(oid string) *gosmi.SmiNode {
	c.lock.Lock()
	defer c.lock.Unlock()

	node, ok := c.cacheMap[oid]
	if !ok {
		return nil
	}

	return node
}

func (c *oidCache) Add(oid string, node *gosmi.SmiNode) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cacheMap[oid] = node
}
