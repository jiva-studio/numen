// Package account lets a plugin with the permission identity ask the host to sign with the key of this
// device, and read what is public of the identity of this device. The key never leaves the host: the plugin
// sends a message and gets a signature, made under the id of the plugin, so a signature a plugin gets
// is never valid for another plugin.
package account

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

const (
	maxMessageSize         = 4096
	maxSealedData          = 32 << 10
	maxSealedOverhead      = 256
	noIdentityReason       = "NO_IDENTITY"
	cannotOpenReason       = "CANNOT_OPEN"
	noPendingRestoreReason = "NO_PENDING_RESTORE"
	invalidChainReason     = "INVALID_CHAIN"
)

var (
	// ErrNoIdentity means this device holds no identity yet.
	ErrNoIdentity = errors.New("no identity on this device")
	// ErrCannotOpen means sealed data does not open for this plugin, context and account.
	ErrCannotOpen = errors.New("the sealed data cannot be opened")
	// ErrNoPendingRestore means this device has no restore that waits for an account chain.
	ErrNoPendingRestore = errors.New("no restore is pending on this device")
	// ErrInvalidChain means the chain does not verify or does not belong to the recovery words.
	ErrInvalidChain = errors.New("the account chain is not valid for the recovery words")
	errMessageSize  = errors.New("the message to sign has 1 to 4096 bytes")
	errDataSize     = errors.New("the data to seal has at most 32 KiB")
)

// PublicIdentity is what is public of the identity of this device.
//
//numen:api
type PublicIdentity struct {
	// AccountFingerprint names the account, in groups of four hexadecimal digits.
	AccountFingerprint string
	// DeviceFingerprint names this device.
	DeviceFingerprint string
	// DeviceSignKey is the Ed25519 public key of this device, 32 bytes.
	DeviceSignKey []byte
	// Chain is the signed device chain of the account.
	Chain []byte
}

// Client asks the host for the identity of this device.
type Client struct {
	rpc numenv1connect.HostAccountServiceClient
}

// NewClient returns the client of the host at the endpoint, sending its calls with the default HTTP client.
func NewClient(endpoint hostcall.Endpoint) *Client {
	return newClientOver(transport.HTTPClient(), endpoint)
}

func newClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewHostAccountServiceClient(httpClient, endpoint.BaseURL, transport.Options(endpoint)...)}
}

// SignAsDevice returns the signature of the key of this device over the message, 64 bytes, made under the
// id of this plugin. The message has 1 to 4096 bytes. It returns ErrNoIdentity where the device holds none.
func (c *Client) SignAsDevice(ctx context.Context, message []byte) ([]byte, error) {
	if len(message) == 0 || len(message) > maxMessageSize {
		return nil, errMessageSize
	}
	response, err := c.rpc.SignAsDevice(ctx, connect.NewRequest(&v1.SignAsDeviceRequest{Message: message}))
	if err != nil {
		return nil, readError(err)
	}
	return response.Msg.GetSignature(), nil
}

// GetPublicIdentity returns what is public of the identity of this device. It returns ErrNoIdentity where
// the device holds none.
func (c *Client) GetPublicIdentity(ctx context.Context) (PublicIdentity, error) {
	response, err := c.rpc.GetPublicIdentity(ctx, connect.NewRequest(&v1.GetPublicIdentityRequest{}))
	if err != nil {
		return PublicIdentity{}, readError(err)
	}
	return PublicIdentity{
		AccountFingerprint: response.Msg.GetAccountFingerprint(),
		DeviceFingerprint:  response.Msg.GetDeviceFingerprint(),
		DeviceSignKey:      response.Msg.GetDeviceSignKey(),
		Chain:              response.Msg.GetChain(),
	}, nil
}

// PendingRestore is a restore of recovery words that waits for the account chain of the server. It holds public
// keys only.
//
//numen:api
type PendingRestore struct {
	// IsPending is false when no restore waits.
	IsPending bool
	// RootSignKey is the Ed25519 public key of the recovery signing key, 32 bytes.
	RootSignKey []byte
	// DeviceSignKey is the Ed25519 public key of the device that joins, 32 bytes.
	DeviceSignKey []byte
}

// SealData encrypts plain with a key only this plugin can derive from the account keys of this device, bound to
// context. Any device of the account opens it. Plain has at most 32 KiB. It returns ErrNoIdentity where the device
// holds none.
func (c *Client) SealData(ctx context.Context, dataContext, plain []byte) ([]byte, error) {
	if len(plain) > maxSealedData {
		return nil, errDataSize
	}
	response, err := c.rpc.SealData(ctx, connect.NewRequest(&v1.SealDataRequest{Context: dataContext, Plain: plain}))
	if err != nil {
		return nil, readError(err)
	}
	return response.Msg.GetSealed(), nil
}

// OpenData decrypts what SealData sealed for this plugin and the same context. It returns ErrCannotOpen for bytes
// that do not open, and ErrNoIdentity where the device holds no identity.
func (c *Client) OpenData(ctx context.Context, dataContext, sealed []byte) ([]byte, error) {
	if len(sealed) > maxSealedData+maxSealedOverhead {
		return nil, errDataSize
	}
	response, err := c.rpc.OpenData(ctx, connect.NewRequest(&v1.OpenDataRequest{Context: dataContext, Sealed: sealed}))
	if err != nil {
		return nil, readError(err)
	}
	return response.Msg.GetPlain(), nil
}

// GetPendingRestore returns the public keys of the restore that waits for an account chain.
func (c *Client) GetPendingRestore(ctx context.Context) (PendingRestore, error) {
	response, err := c.rpc.GetPendingRestore(ctx, connect.NewRequest(&v1.GetPendingRestoreRequest{}))
	if err != nil {
		return PendingRestore{}, readError(err)
	}
	return PendingRestore{
		IsPending:     response.Msg.GetIsPending(),
		RootSignKey:   response.Msg.GetRootSignKey(),
		DeviceSignKey: response.Msg.GetDeviceSignKey(),
	}, nil
}

// SignAsRecovery returns the signature, 64 bytes, of the recovery key of the pending restore over the message,
// made under the id of this plugin. The message has 1 to 4096 bytes. It returns ErrNoPendingRestore where no
// restore is pending.
func (c *Client) SignAsRecovery(ctx context.Context, message []byte) ([]byte, error) {
	if len(message) == 0 || len(message) > maxMessageSize {
		return nil, errMessageSize
	}
	response, err := c.rpc.SignAsRecovery(ctx, connect.NewRequest(&v1.SignAsRecoveryRequest{Message: message}))
	if err != nil {
		return nil, readError(err)
	}
	return response.Msg.GetSignature(), nil
}

// CompleteRestore finishes the pending restore with the chain the server stores for the account, or with no chain
// where the server has no such account. It returns ErrNoPendingRestore where none is pending and ErrInvalidChain
// for a chain that does not verify or does not belong to the recovery words.
func (c *Client) CompleteRestore(ctx context.Context, chain []byte) error {
	if _, err := c.rpc.CompleteRestore(ctx, connect.NewRequest(&v1.CompleteRestoreRequest{Chain: chain})); err != nil {
		return readError(err)
	}
	return nil
}

func readError(err error) error {
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return err
	}
	switch readReason(failure) {
	case noIdentityReason:
		return ErrNoIdentity
	case cannotOpenReason:
		return ErrCannotOpen
	case noPendingRestoreReason:
		return ErrNoPendingRestore
	case invalidChainReason:
		return ErrInvalidChain
	}
	return err
}

func readReason(failure *connect.Error) string {
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if info, isInfo := value.(*errdetails.ErrorInfo); isInfo {
			return info.GetReason()
		}
	}
	return ""
}
