package storage

import (
	"github.com/gohornet/hornet/pkg/model/milestone"
	"github.com/iotaledger/hive.go/syncutils"
)

func CachedMessageCaller(handler interface{}, params ...interface{}) {
	handler.(func(msIndex milestone.Index, cachedMessage *CachedMessage, cachedChildren CachedChildren))(params[0].(milestone.Index), params[1].(*CachedMessage).Retain(), params[2].(CachedChildren).Retain())
}

func CachedMetadataCaller(handler interface{}, params ...interface{}) {
	handler.(func(msIndex milestone.Index, cachedMetadata *CachedMetadata))(params[0].(milestone.Index), params[1].(*CachedMetadata).Retain())
}

// NewTangleCache creates a new NewTangleCache instance.
func NewTangleCache() *TangleCache {
	return &TangleCache{
		cachedMsgs:        make(map[milestone.Index][]*CachedMessage),
		cachedMsgMetas:    make(map[milestone.Index][]*CachedMetadata),
		cachedChildren:    make(map[milestone.Index][]*CachedChild),
		minMilestoneIndex: 0,
	}
}

// TangleCache holds an object storage reference to objects, based on their milestone index.
// This is used to keep the latest cone of the tangle in the cache, without relying on cacheTimes at all.
type TangleCache struct {
	syncutils.Mutex

	cachedMsgs     map[milestone.Index][]*CachedMessage
	cachedMsgMetas map[milestone.Index][]*CachedMetadata
	cachedChildren map[milestone.Index][]*CachedChild

	minMilestoneIndex milestone.Index
}

func (c *TangleCache) SetMinMilestoneIndex(msIndex milestone.Index) {
	c.Lock()
	defer c.Unlock()
	c.minMilestoneIndex = msIndex
}

func (c *TangleCache) AddCachedMessage(msIndex milestone.Index, cachedMsg *CachedMessage) {
	c.Lock()
	defer c.Unlock()
	defer cachedMsg.Release(true)

	if msIndex < c.minMilestoneIndex {
		return
	}

	if _, exists := c.cachedMsgs[msIndex]; !exists {
		c.cachedMsgs[msIndex] = []*CachedMessage{}
	}
	c.cachedMsgs[msIndex] = append(c.cachedMsgs[msIndex], cachedMsg.Retain())
}

func (c *TangleCache) AddCachedMetadata(msIndex milestone.Index, cachedMsgMeta *CachedMetadata) {
	c.Lock()
	defer c.Unlock()
	defer cachedMsgMeta.Release(true)

	if msIndex < c.minMilestoneIndex {
		return
	}

	if _, exists := c.cachedMsgMetas[msIndex]; !exists {
		c.cachedMsgMetas[msIndex] = []*CachedMetadata{}
	}
	c.cachedMsgMetas[msIndex] = append(c.cachedMsgMetas[msIndex], cachedMsgMeta.Retain())
}

func (c *TangleCache) AddCachedChildren(msIndex milestone.Index, cachedChildren CachedChildren) {
	c.Lock()
	defer c.Unlock()
	defer cachedChildren.Release(true)

	if msIndex < c.minMilestoneIndex {
		return
	}

	if _, exists := c.cachedChildren[msIndex]; !exists {
		c.cachedChildren[msIndex] = []*CachedChild{}
	}
	c.cachedChildren[msIndex] = append(c.cachedChildren[msIndex], cachedChildren.Retain()...)
}

func (c *TangleCache) ReleaseCachedMessages(msIndex milestone.Index, forceRelease ...bool) {
	c.Lock()
	defer c.Unlock()

	for index, cachedMessages := range c.cachedMsgs {
		if index > msIndex {
			// only release entries that belong to older milestones
			continue
		}

		for _, cachedMsg := range cachedMessages {
			cachedMsg.Release(forceRelease...)
		}
		delete(c.cachedMsgs, msIndex)
	}
}

func (c *TangleCache) ReleaseCachedMetadata(msIndex milestone.Index, forceRelease ...bool) {
	c.Lock()
	defer c.Unlock()

	for index, cachedMsgMetas := range c.cachedMsgMetas {
		if index > msIndex {
			// only release entries that belong to older milestones
			continue
		}

		for _, cachedMsgMeta := range cachedMsgMetas {
			cachedMsgMeta.Release(forceRelease...)
		}
		delete(c.cachedMsgMetas, msIndex)
	}
}

func (c *TangleCache) ReleaseCachedChildren(msIndex milestone.Index, forceRelease ...bool) {
	c.Lock()
	defer c.Unlock()

	for index, cachedChildren := range c.cachedChildren {
		if index > msIndex {
			// only release entries that belong to older milestones
			continue
		}

		for _, cachedChild := range cachedChildren {
			cachedChild.Release(forceRelease...)
		}
		delete(c.cachedChildren, msIndex)
	}
}

// Cleanup releases all the cached objects that have been used.
// This MUST be called by the user at the end.
func (c *TangleCache) Cleanup() {
	c.Lock()
	defer c.Unlock()

	// release all messages at the end
	for _, cachedMessages := range c.cachedMsgs {
		for _, cachedMsg := range cachedMessages {
			cachedMsg.Release(true)
		}
	}

	// release all msg metadata at the end
	for _, cachedMsgMetas := range c.cachedMsgMetas {
		for _, cachedMsgMeta := range cachedMsgMetas {
			cachedMsgMeta.Release(true)
		}
	}

	// release all children at the end
	for _, cachedChildren := range c.cachedChildren {
		for _, cachedChild := range cachedChildren {
			cachedChild.Release(true)
		}
	}
}
