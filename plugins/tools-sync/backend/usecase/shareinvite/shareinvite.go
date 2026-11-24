// Package shareinvite shares a folder by a link: the sharer makes a link over a channel of the relay, the person who
// pastes it joins the channel, both show the same code, and the sharer approves the request, which adds the person as a
// reader of the folder and hands it the invitation over the same channel.
package shareinvite

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	// MaxLifetime is how long a link works, at most. The relay ends a channel sooner when its own lifetime is shorter.
	MaxLifetime = 24 * time.Hour
	waitFor     = 10 * time.Second
	retryAfter  = time.Second
	taskBuffer  = 16
	idSize      = 16
	usedMarker  = "numen-share-used"
)

// Deps are what the use cases of sharing by link need.
type Deps struct {
	Relay     ports.Relay
	Health    ports.Health
	Clock     ports.Clock
	Identity  ports.Identity
	Device    ports.Device
	Sharing   ports.FolderSharing
	Receiving ports.InvitationReceiving
	Random    io.Reader
	Server    string
	Hub       *Hub
}

// Hub keeps the links and requests of the plugin that run: it lives as long as the plugin does, and holds nothing
// that is not lost when the plugin stops.
type Hub struct {
	mu     sync.Mutex
	offers map[string]*offer
	joins  map[string]*join
	tasks  chan func(context.Context)
}

// NewHub returns a hub with no link and no request.
func NewHub() *Hub {
	return &Hub{mu: sync.Mutex{}, offers: map[string]*offer{}, joins: map[string]*join{}, tasks: make(chan func(context.Context), taskBuffer)}
}

func (h *Hub) add(ctx context.Context, task func(context.Context)) error {
	select {
	case h.tasks <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// offer is a link the sharer made. request is set when a person joined it and sent its identity.
type offer struct {
	id        []byte
	vault     domain.VaultID
	folder    string
	name      string
	expiresAt time.Time
	displayer *identity.LinkDisplayer
	channel   *identity.LinkChannel
	request   *pending
	isDone    bool
}

// pending is a person who joined a link and waits for the sharer to answer.
type pending struct {
	chain       []byte
	name        string
	fingerprint string
	code        string
	isBusy      bool
}

// Request is a person who wants to read a folder, as the sharer lists it.
type Request struct {
	ID          string
	VaultID     domain.VaultID
	Folder      string
	DeviceName  string
	Fingerprint string
	Code        string
}
