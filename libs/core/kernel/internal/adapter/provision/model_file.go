package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (p *FileProvisioner) getCachedFile(req manifest.DependencyRequirement, dir, pin string) (plugin.ResolvedDependency, bool) {
	if pin != "" {
		return readCachedDependency(req, dir, pin)
	}
	_, sum, err := readReadyMarker(dir)
	digest, errDigest := hex.DecodeString(sum)
	if err != nil || errDigest != nil || len(digest) != sha256.Size {
		return plugin.ResolvedDependency{}, false
	}
	return readCachedDependency(req, p.getModelContentDir(sum), sum)
}

func (p *FileProvisioner) getModelContentDir(sum string) string {
	return getDependencyDir(p.cfg.Dir, string(manifest.DependencyKindModel), sum, "")
}

func (p *FileProvisioner) storeModelFile(ctx context.Context, req manifest.DependencyRequirement, source, primary, sum string) (_ plugin.ResolvedDependency, err error) {
	dir := p.getModelContentDir(sum)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	release, err := lockFile(ctx, dir+".lock")
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	defer func() { err = errors.Join(err, release()) }()
	resolved, err := installContent(ctx, req, dir, primary, sum)
	if err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if err := writeReadyMarker(ctx, source, filepath.Base(resolved.PrimaryPath), sum); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	return resolved, nil
}

func installContent(ctx context.Context, req manifest.DependencyRequirement, dir, primary, sum string) (plugin.ResolvedDependency, error) {
	if resolved, exists := readCachedDependency(req, dir, sum); exists {
		return resolved, os.Remove(primary)
	}
	dest := filepath.Join(dir, filepath.Base(primary))
	if err := os.Rename(primary, dest); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	if err := writeReadyMarker(ctx, dir, filepath.Base(dest), sum); err != nil {
		return plugin.ResolvedDependency{}, err
	}
	return newResolvedDependency(req, dir, dest), nil
}
