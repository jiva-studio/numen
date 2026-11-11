package serving

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/providers"
)

func (s *Server) refreshModelInfoLocked() {
	s.refreshModelIdentityLocked()
	s.modelInfo.ChunkTokens = chunking.ClampTokens(s.chunkTokens, s.modelInfo.MaxTokens)
}

func (s *Server) refreshModelIdentityLocked() {
	if s.isMock {
		return
	}
	keep := s.modelInfo
	if s.provider != providers.Local && s.remoteModel != "" {
		profile := s.profiles.GetRemote(s.remoteModel)
		s.modelInfo = engine.ModelInfo{
			Name: s.remoteModel, Dimensions: profile.Dimensions, MaxTokens: profile.GetTokens(), Pooling: profile.Pooling,
			From: keep.From, Concurrency: keep.Concurrency, FloorHint: profile.FloorHint, MinChunkLength: keep.MinChunkLength,
			ChunkTokens: chunking.ClampTokens(s.chunkTokens, profile.GetTokens()),
			Handling:    profile.GetHandling(),
		}
		s.env.GetOverrides().ApplyToInfo(&s.modelInfo)
		return
	}
	s.modelInfo = s.localInfo
	s.modelInfo.Concurrency = keep.Concurrency
	s.modelInfo.MinChunkLength = keep.MinChunkLength
}

func (s *Server) updateLocalModelLocked(model string) {
	if s.isMock {
		return
	}
	info, local, err := s.resolveLocalModel(model)
	if s.profilesErr != nil {
		err = s.profilesErr
	}
	s.localErr = err
	if err == nil {
		s.localInfo, s.localModel, s.localName = info, local, info.Name
	} else {
		s.localName = model
	}
	if key := getLocalKey(info, local, err); key != s.localKey {
		s.localKey = key
		s.loader.ResetLocked()
	}
}

func getLocalKey(info engine.ModelInfo, local engine.LocalModel, err error) string {
	if err != nil {
		return "error"
	}
	return strings.Join([]string{
		info.Name, strconv.Itoa(info.Dimensions), strconv.Itoa(info.MaxTokens), info.Pooling,
		info.Handling.Hash(info.MaxTokens), strings.Join(info.Handling.ExtraInputs, ","),
		local.Name, local.Dir, local.File, local.Revision,
	}, "|")
}

func (s *Server) resolveLocalModel(model string) (engine.ModelInfo, engine.LocalModel, error) {
	overrides := s.env.GetOverrides()
	if model == "" {
		model = overrides.Model
	}
	if model == "" || s.profiles.IsRemote(model) {
		model = engine.DefaultModel
	}
	profile, err := s.profiles.GetLocal(model)
	if err != nil {
		return engine.ModelInfo{}, engine.LocalModel{}, err
	}
	info, local := engine.NewConfig(profile)
	overrides.ApplyToInfo(&info)
	overrides.ApplyToLocal(&local)
	return info, local, nil
}
