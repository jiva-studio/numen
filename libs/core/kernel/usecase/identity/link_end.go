package identity

import identitylib "github.com/jiva-studio/numen/libs/shared/identity"

// linkEnd is the end of the link channel on this device: the key exchange of the side that shows the QR code or of
// the side that reads it, and then the channel the exchange made.
type linkEnd interface {
	// accept takes the first message of the other side and makes the channel; reply is what goes back, if anything.
	accept(blob []byte) (reply []byte, err error)
	isOpen() bool
	seal(plain []byte) []byte
	open(blob []byte) ([]byte, error)
}

type linkChannel struct{ channel *identitylib.LinkChannel }

func (c *linkChannel) isOpen() bool { return c.channel != nil }

func (c *linkChannel) seal(plain []byte) []byte { return c.channel.Seal(plain) }

func (c *linkChannel) open(blob []byte) ([]byte, error) { return c.channel.Open(blob) }

// displayerEnd is the end of the side that shows the QR code.
type displayerEnd struct {
	linkChannel
	displayer *identitylib.LinkDisplayer
}

func (e *displayerEnd) accept(blob []byte) ([]byte, error) {
	reply, channel, err := e.displayer.Accept(blob)
	if err != nil {
		return nil, err
	}
	e.channel = channel
	return reply, nil
}

// scannerEnd is the end of the side that reads the QR code.
type scannerEnd struct {
	linkChannel
	scanner *identitylib.LinkScanner
}

func (e *scannerEnd) accept(blob []byte) ([]byte, error) {
	channel, err := e.scanner.Accept(blob)
	if err != nil {
		return nil, err
	}
	e.channel = channel
	return nil, nil
}
