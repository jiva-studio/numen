package epub

import (
	"archive/zip"
)

const encryptionPath = "META-INF/encryption.xml"

// readEncryptedPaths are the archive entries META-INF/encryption.xml names. A book
// carrying none has no such file.
//
// What was done to an entry is not read: which entries the file names is what
// separates a book that reads from one that does not.
func readEncryptedPaths(files map[string]*zip.File) map[string]bool {
	raw, ok := contents(files[encryptionPath])
	if !ok {
		return nil
	}
	var document struct {
		References []struct {
			URI string `xml:"URI,attr"`
		} `xml:"EncryptedData>CipherData>CipherReference"`
	}
	if err := decodeXML(raw, &document); err != nil {
		return nil
	}

	locked := map[string]bool{}
	for _, reference := range document.References {
		// A cipher reference is written against the root of the archive.
		if at := resolve("", reference.URI); at != "" {
			locked[at] = true
		}
	}
	return locked
}
