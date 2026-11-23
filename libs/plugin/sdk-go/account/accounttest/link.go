package accounttest

import "errors"

// ShowLink makes the host start a link in mode show, as the window of a host does.
func (f *Fake) ShowLink() error { return errors.New("not implemented") }
