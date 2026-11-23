package identity

import (
	"context"
	"errors"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

func errNotImplemented() error { return errors.New("not implemented") }

// WithChains returns the deps over the chain file of this device.
func (d Deps) WithChains(chains ChainStore) Deps { return d }

type StartLink struct{ deps Deps }

func NewStartLink(d Deps) *StartLink { return &StartLink{deps: d} }

func (u *StartLink) Execute(ctx context.Context, role identitydomain.LinkRole, mode identitydomain.LinkMode, scanned string) (identitydomain.Link, error) {
	return identitydomain.Link{}, errNotImplemented()
}

type GetLink struct{ deps Deps }

func NewGetLink(d Deps) *GetLink { return &GetLink{deps: d} }

func (u *GetLink) Execute(ctx context.Context) (identitydomain.Link, error) {
	return identitydomain.Link{}, errNotImplemented()
}

type DecideLink struct{ deps Deps }

func NewDecideLink(d Deps) *DecideLink { return &DecideLink{deps: d} }

func (u *DecideLink) Execute(ctx context.Context, approve bool) error { return errNotImplemented() }

type EnterLinkCode struct{ deps Deps }

func NewEnterLinkCode(d Deps) *EnterLinkCode { return &EnterLinkCode{deps: d} }

func (u *EnterLinkCode) Execute(ctx context.Context, code string) error { return errNotImplemented() }

type ConfirmJoin struct{ deps Deps }

func NewConfirmJoin(d Deps) *ConfirmJoin { return &ConfirmJoin{deps: d} }

func (u *ConfirmJoin) Execute(ctx context.Context, approve bool) error { return errNotImplemented() }

type CancelLink struct{ deps Deps }

func NewCancelLink(d Deps) *CancelLink { return &CancelLink{deps: d} }

func (u *CancelLink) Execute(ctx context.Context) error { return errNotImplemented() }

type ListDevices struct{ deps Deps }

func NewListDevices(d Deps) *ListDevices { return &ListDevices{deps: d} }

func (u *ListDevices) Execute(ctx context.Context) ([]identitydomain.Device, error) {
	return nil, errNotImplemented()
}

type RevokeDevice struct{ deps Deps }

func NewRevokeDevice(d Deps) *RevokeDevice { return &RevokeDevice{deps: d} }

func (u *RevokeDevice) Execute(ctx context.Context, fingerprint string) error {
	return errNotImplemented()
}

type GetLinkWork struct{ deps Deps }

func NewGetLinkWork(d Deps) *GetLinkWork { return &GetLinkWork{deps: d} }

func (u *GetLinkWork) Execute(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error) {
	return identitydomain.LinkWork{}, errNotImplemented()
}

type SetLinkChannel struct{ deps Deps }

func NewSetLinkChannel(d Deps) *SetLinkChannel { return &SetLinkChannel{deps: d} }

func (u *SetLinkChannel) Execute(ctx context.Context, id []byte) error { return errNotImplemented() }

type PutLinkInbound struct{ deps Deps }

func NewPutLinkInbound(d Deps) *PutLinkInbound { return &PutLinkInbound{deps: d} }

func (u *PutLinkInbound) Execute(ctx context.Context, blob []byte) error { return errNotImplemented() }

type GetChainPublish struct{ deps Deps }

func NewGetChainPublish(d Deps) *GetChainPublish { return &GetChainPublish{deps: d} }

func (u *GetChainPublish) Execute(ctx context.Context) (identitydomain.ChainPublish, error) {
	return identitydomain.ChainPublish{}, errNotImplemented()
}

type PutChainPublished struct{ deps Deps }

func NewPutChainPublished(d Deps) *PutChainPublished { return &PutChainPublished{deps: d} }

func (u *PutChainPublished) Execute(ctx context.Context, accepted bool, latest []byte) error {
	return errNotImplemented()
}

type AdoptChain struct{ deps Deps }

func NewAdoptChain(d Deps) *AdoptChain { return &AdoptChain{deps: d} }

func (u *AdoptChain) Execute(ctx context.Context, chain []byte) error { return errNotImplemented() }
