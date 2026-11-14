package download

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
)

// mockVideo replaces yt-dlp on the video sites: it publishes the captions of a short
// clip and copies a clip of silent audio written here, and makes no network request.
type mockVideo struct{}

// NewMock is a downloader that answers a video site with the mock and reads every other page as
// itself.
func NewMock(web Web) *Downloader {
	return &Downloader{providers: []Provider{mockVideo{}, NewPages(web)}}
}

// mockSeconds is how long the clip runs.
const mockSeconds = 3

func getMockCues() []ytdlp.Cue {
	return []ytdlp.Cue{
		{Text: "The tide comes in twice a day.", From: 0, To: 1000},
		{Text: "The mill pond fills behind the gate.", From: 1000, To: 2000},
		{Text: "On the ebb the wheel turns.", From: 2000, To: 3000},
	}
}

func (mockVideo) CanHandle(address string) bool { return isVideoSite(address) }

func (mockVideo) GetDownloadModel(string) fetch.Model {
	return fetch.Model{Tool: "mock", Version: "", Producer: producerCaptions}
}

func (mockVideo) GetMetadata(context.Context, string) (fetch.Metadata, error) {
	return fetch.Metadata{
		Title: "Tidal mill at work", Bytes: int64(len(writeQuietWAV())), Length: mockSeconds * 1000,
		Language: "en", Captions: []string{"en"}, IsAutomatic: nil,
	}, nil
}

func (mockVideo) GetText(context.Context, string, fetch.PreferredCaptions) (fetch.Text, error) {
	return fetch.Text{
		Producer: producerCaptions, Title: "Tidal mill at work", Length: mockSeconds * 1000,
		Cues: getMockCues(), Prose: "",
	}, nil
}

func (mockVideo) Download(_ context.Context, _ string, _ fetch.Media, into io.Writer) (fetch.Copy, error) {
	if _, err := into.Write(writeQuietWAV()); err != nil {
		return fetch.Copy{}, err
	}
	return fetch.Copy{MediaType: "audio/wav", Extension: ".wav"}, nil
}

func writeQuietWAV() []byte {
	const rate = 8000
	samples := rate * mockSeconds
	out := make([]byte, 44+samples)
	copy(out[0:], "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(36+samples))
	copy(out[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(out[16:], 16)
	binary.LittleEndian.PutUint16(out[20:], 1)
	binary.LittleEndian.PutUint16(out[22:], 1)
	binary.LittleEndian.PutUint32(out[24:], rate)
	binary.LittleEndian.PutUint32(out[28:], rate)
	binary.LittleEndian.PutUint16(out[32:], 1)
	binary.LittleEndian.PutUint16(out[34:], 8)
	copy(out[36:], "data")
	binary.LittleEndian.PutUint32(out[40:], uint32(samples))
	for i := 44; i < len(out); i++ {
		out[i] = 128
	}
	return out
}
