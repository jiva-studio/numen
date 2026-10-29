// Package api declares the JSON messages that the backend of the flashcards-studio plugin
// exchanges with its frontend under the `numen.flashcards` service.
//
// A field is named in lowerCamelCase on the wire. An enumeration is a string whose zero value is
// the empty string. A scalar field is always present; a message-typed field or a scalar behind a
// pointer is absent when it has no value. A response that can fail on a note carries the failure
// in its `error` field; the other failures are API errors.
package api
