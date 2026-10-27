package serviceproxy

import "slices"

// GetFrameOrigins lists the origins the pages of an enabled plugin frame, and none for a plugin that
// is not enabled.
func (p *Proxy) GetFrameOrigins(pluginID string) []string {
	found, enabled := p.host.GetPlugin(pluginID)
	if !enabled {
		return nil
	}
	return slices.Clone(found.Manifest.Frames)
}

// ListFrameOrigins lists the origins the pages of every enabled plugin frame, each once, in the order
// of the plugins' ids.
func (p *Proxy) ListFrameOrigins() []string {
	var out []string
	for _, info := range p.host.GetPluginInfos() {
		for _, origin := range p.GetFrameOrigins(info.ID) {
			if !slices.Contains(out, origin) {
				out = append(out, origin)
			}
		}
	}
	return out
}
