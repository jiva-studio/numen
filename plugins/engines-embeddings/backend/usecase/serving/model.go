package serving

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/providers"
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
		s.modelInfo = model.ModelInfo{
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

func (s *Server) updateLocalModelLocked(name string) {
	if s.isMock {
		return
	}
	info, local, err := s.resolveLocalModel(name)
	if s.profilesErr != nil {
		err = s.profilesErr
	}
	s.localErr = err
	if err == nil {
		s.localInfo, s.localModel, s.localName = info, local, info.Name
	} else {
		s.localName = name
	}
	if key := getLocalKey(info, local, err); key != s.localKey {
		s.localKey = key
		s.loader.ResetLocked()
	}
}

func getLocalKey(info model.ModelInfo, local model.LocalModel, err error) string {
	if err != nil {
		return "error"
	}
	return strings.Join([]string{
		info.Name, strconv.Itoa(info.Dimensions), strconv.Itoa(info.MaxTokens), info.Pooling,
		info.Handling.Hash(info.MaxTokens), strings.Join(info.Handling.ExtraInputs, ","),
		local.Name, local.Dir, local.File, local.Revision,
	}, "|")
}

func (s *Server) resolveLocalModel(name string) (model.ModelInfo, model.LocalModel, error) {
	overrides := s.env.GetOverrides()
	if name == "" {
		name = overrides.Model
	}
	if name == "" || s.profiles.IsRemote(name) {
		name = model.DefaultModel
	}
	profile, err := s.profiles.GetLocal(name)
	if err != nil {
		return model.ModelInfo{}, model.LocalModel{}, err
	}
	info, local := model.NewConfig(profile)
	overrides.ApplyToInfo(&info)
	overrides.ApplyToLocal(&local)
	return info, local, nil
}
