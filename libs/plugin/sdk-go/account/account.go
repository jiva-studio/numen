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
	maxMessageSize   = 4096
	noIdentityReason = "NO_IDENTITY"
)

var (
	// ErrNoIdentity means this device holds no identity yet.
	ErrNoIdentity  = errors.New("no identity on this device")
	errMessageSize = errors.New("the message to sign has 1 to 4096 bytes")
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
	return newClientOver(transport.GetHTTPClient(), endpoint)
}

func newClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewHostAccountServiceClient(httpClient, endpoint.BaseURL, transport.BuildOptions(endpoint)...)}
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

func readError(err error) error {
	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition {
		return err
	}
	for _, detail := range failure.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}
		if info, isInfo := value.(*errdetails.ErrorInfo); isInfo && info.GetReason() == noIdentityReason {
			return ErrNoIdentity
		}
	}
	return err
}
