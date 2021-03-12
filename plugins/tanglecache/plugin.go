package tanglecache

import (
	"go.uber.org/dig"

	"github.com/gohornet/hornet/pkg/model/milestone"
	"github.com/gohornet/hornet/pkg/model/storage"
	"github.com/gohornet/hornet/pkg/node"
	"github.com/gohornet/hornet/pkg/shutdown"
	"github.com/gohornet/hornet/pkg/tangle"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
)

func init() {
	Plugin = &node.Plugin{
		Status: node.Enabled,
		Pluggable: node.Pluggable{
			Name:      "TangleCache",
			DepsFunc:  func(cDeps dependencies) { deps = cDeps },
			Configure: configure,
			Run:       run,
		},
	}
}

var (
	Plugin *node.Plugin
	log    *logger.Logger
	deps   dependencies

	tangleCache *storage.TangleCache

	onCachedMessage                  *events.Closure
	onCachedMetadata                 *events.Closure
	onConfirmedMilestoneIndexChanged *events.Closure
)

type dependencies struct {
	dig.In
	Storage       *storage.Storage
	Tangle        *tangle.Tangle
	BelowMaxDepth int `name:"belowMaxDepth"`
}

func configure() {
	log = logger.NewLogger(Plugin.Name)
	tangleCache = storage.NewTangleCache()
	configureEvents()
}

func run() {
	Plugin.Daemon().BackgroundWorker("TangleCache", func(shutdownSignal <-chan struct{}) {
		log.Info("Starting TangleCache ... done")
		attachEvents()
		<-shutdownSignal
		log.Info("Stopping TangleCache ...")
		detachEvents()
		tangleCache.Cleanup()
		log.Info("Stopping TangleCache ... done")
	}, shutdown.PriorityTangleCache)
}

func configureEvents() {

	// if the node is unsync, we want to cache all requested messages
	// with the index of the milestone that requested the message. (CachedMessageRequestedAndStored)
	//
	// if the node is sync, we want to cache all messages that become solid
	// with the index of their youngest cone root index. (CachedMessageSolid)
	//
	// we cache the messages here, because they will be used for the milestone confirmation afterwards.
	onCachedMessage = events.NewClosure(func(msIndex milestone.Index, cachedMessage *storage.CachedMessage, cachedChildren storage.CachedChildren) {
		tangleCache.AddCachedMessage(msIndex, cachedMessage)
		tangleCache.AddCachedMetadata(msIndex, cachedMessage.GetCachedMetadata())
		tangleCache.AddCachedChildren(msIndex, cachedChildren)
	})

	// furthermore we also want to cache all metadata that get referenced by
	// a milestone if the node is sync. (CachedMessageReferenced)
	//
	// It's no problem if these metadata were already "cached" because
	// they became solid before, it only increases the reference counter.
	onCachedMetadata = events.NewClosure(func(msIndex milestone.Index, cachedMetadata *storage.CachedMetadata) {
		tangleCache.AddCachedMetadata(msIndex, cachedMetadata)
	})

	// messages are only needed until they are referenced by a milestone.
	// message metadata is needed, until it falls below max depth.
	// children are needed, until they fall below max depth.
	onConfirmedMilestoneIndexChanged = events.NewClosure(func(msIndex milestone.Index) {
		tangleCache.SetMinMilestoneIndex(msIndex)

		tangleCache.ReleaseCachedMessages(msIndex-1, true)
		if msIndex < milestone.Index(deps.BelowMaxDepth) {
			return
		}
		tangleCache.ReleaseCachedMetadata(msIndex-milestone.Index(deps.BelowMaxDepth), true)
		tangleCache.ReleaseCachedChildren(msIndex-milestone.Index(deps.BelowMaxDepth), true)
	})
}

func attachEvents() {
	deps.Storage.Events.CachedMessageRequestedAndStored.Attach(onCachedMessage)
	deps.Tangle.Events.CachedMessageSolid.Attach(onCachedMessage)
	deps.Tangle.Events.CachedMetadataReferenced.Attach(onCachedMetadata)
	deps.Tangle.Events.ConfirmedMilestoneIndexChanged.Attach(onConfirmedMilestoneIndexChanged)
}

func detachEvents() {
	deps.Storage.Events.CachedMessageRequestedAndStored.Detach(onCachedMessage)
	deps.Tangle.Events.CachedMessageSolid.Detach(onCachedMessage)
	deps.Tangle.Events.CachedMetadataReferenced.Detach(onCachedMetadata)
	deps.Tangle.Events.ConfirmedMilestoneIndexChanged.Detach(onConfirmedMilestoneIndexChanged)
}
