package container

import (
	"archive/zip"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/archive"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/href"
)

const (
	containerPath = "META-INF/container.xml"
	mediaPackage  = "application/oebps-package+xml"
)

// ErrNoContainer: the archive holds no container naming a package document.
var ErrNoContainer = errors.New("epub: no META-INF/container.xml")

// ReadPackagePath reads the container for the name of the package document.
func ReadPackagePath(files map[string]*zip.File) (string, error) {
	raw, ok := archive.ReadEntry(files[containerPath])
	if !ok {
		return "", ErrNoContainer
	}
	var container struct {
		Rootfiles []struct {
			Path      string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := archive.DecodeXML(raw, &container); err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoContainer, err)
	}

	first := ""
	for _, root := range container.Rootfiles {
		if root.Path == "" {
			continue
		}
		pos := href.Resolve("", root.Path)
		if root.MediaType == mediaPackage {
			return pos, nil
		}
		if first == "" {
			first = pos
		}
	}
	if first == "" {
		return "", fmt.Errorf("%w: it names no rootfile", ErrNoContainer)
	}
	return first, nil
}
