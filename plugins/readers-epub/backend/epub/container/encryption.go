package container

import (
	"archive/zip"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/archive"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
)

const encryptionPath = "META-INF/encryption.xml"

// ReadEncryptedPaths are the archive entries META-INF/encryption.xml names. A book
// carrying none has no such file.
//
// What was done to an entry is not read: which entries the file names is what
// separates a book that reads from one that does not.
func ReadEncryptedPaths(files map[string]*zip.File) map[string]bool {
	raw, _ := archive.ReadEntry(files[encryptionPath])
	var document struct {
		References []struct {
			URI string `xml:"URI,attr"`
		} `xml:"EncryptedData>CipherData>CipherReference"`
	}
	if err := archive.DecodeXML(raw, &document); err != nil {
		return nil
	}

	locked := map[string]bool{}
	for _, reference := range document.References {
		// A cipher reference is written against the root of the archive.
		if pos := href.Resolve("", reference.URI); pos != "" {
			locked[pos] = true
		}
	}
	return locked
}
