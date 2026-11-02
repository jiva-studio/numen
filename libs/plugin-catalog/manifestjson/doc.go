// Package manifestjson reads manifest.json into a manifest.Manifest. It is the one place that decodes
// the file: the fields come from the Manifest message of the schema in libs/protocol, and the model
// of the manifest package is built from that message field by field.
package manifestjson
