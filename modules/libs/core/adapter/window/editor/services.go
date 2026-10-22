package editor

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Services is the handlers a window answers through, one for each service of the
// schema, and the pieces beside them that are not a service.
//
// A handler holds the use cases it calls and the State it reads the vault
// from. The composition fills each in once the use cases are built.
type Services struct {
	state *State

	Vault      *vaultHandler
	Workspace  *workspaceHandler
	File       *fileHandler
	Note       *noteHandler
	Search     *searchHandler
	Agent      *agentHandler
	Vaults     *vaultsHandler
	Settings   *settingsHandler
	Artifact   *artifactHandler
	Transcript *transcriptHandler
	Article    *articleHandler
	Ocr        *ocrHandler
	Document   *documentHandler
	Recording  *recordingHandler
	Book       *bookHandler

	// Asset answers the bytes of a file at an address of its own.
	Asset *assetHandler

	// Themes are the stylesheets the window may be dressed in. They belong to
	// the installation, so they arrive here from whatever put the window
	// together. A build put together without a catalogue serves no ThemeService.
	Themes numenv1connect.ThemeServiceHandler

	// PluginServices answers the calls for the services plugins serve to the windows.
	PluginServices PluginServices

	// Plugin answers questions about the plugins installed in the application.
	Plugin numenv1connect.PluginServiceHandler

	// getPluginDir returns the root directory of an installed plugin by ID, or false if not found.
	getPluginDir func(id string) (string, bool)

	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback

	// files finds the file a question about an artifact is about, and server
	// answers with the file itself on the socket a recording is played from.
	files  *artifactFiles
	server *fileServer
}

// createServices is every handler of a window over this state, each holding
// nothing yet that needs a vault.
func createServices(state *State) *Services {
	files := &artifactFiles{state: state}
	document := &documentHandler{state: state}
	book := &bookHandler{state: state}
	return &Services{
		state:      state,
		Vault:      &vaultHandler{state: state},
		Workspace:  &workspaceHandler{state: state},
		File:       &fileHandler{state: state},
		Note:       &noteHandler{state: state},
		Search:     &searchHandler{state: state},
		Agent:      &agentHandler{state: state},
		Vaults:     &vaultsHandler{state: state},
		Settings:   &settingsHandler{},
		Artifact:   &artifactHandler{state: state, files: files},
		Transcript: &transcriptHandler{state: state, files: files},
		Article:    &articleHandler{files: files},
		Ocr:        &ocrHandler{files: files},
		Document:   document,
		Recording:  &recordingHandler{files: files},
		Book:       book,
		Asset:      &assetHandler{document: document, book: book},
		Plugin:     numenv1connect.UnimplementedPluginServiceHandler{},
		files:      files,
		server:     &fileServer{state: state, files: files},
	}
}

// openPlaying names the socket a recording is played from.
func (s *Services) openPlaying(playing *Loopback) {
	s.Playing = playing
	s.Recording.playing = playing
}

// readDocuments binds the use cases that read the documents and books of the
// vaults the readers open, through what the viewer holds open.
func (s *Services) readDocuments(readers port.VaultReaders, books port.Book, v *viewer) {
	getDocument := document.NewGetDocument(readers, v)
	getPage := document.NewGetPage(readers, v, v)
	getBookPicture := document.NewGetBookPicture(readers, books, http.DetectContentType)

	s.Document.viewer, s.Document.getDocument, s.Document.getPage = v, &getDocument, &getPage
	getBook, readBookMarkup := document.NewGetBook(books), document.NewReadBookMarkup(books)

	s.Book.viewer, s.Book.readers, s.Book.getBookPicture = v, readers, &getBookPicture
	s.Book.getBook, s.Book.readBookMarkup = &getBook, &readBookMarkup
}

// readFiles binds the use cases that read the files of the vaults the readers open.
func (s *Services) readFiles(readers port.VaultReaders) {
	s.files.getFingerprint = file.NewGetFingerprint(readers)
	s.server.openFile = document.NewOpenFile(readers)
}

// readTexts binds the use cases that read and write what was made from a file
// of the vault, each to the handlers that call it.
func (s *Services) readTexts(texts source.Texts) {
	s.files.getNoteAddress = texts.GetNoteAddress
	s.server.openCopy = texts.OpenCopy

	s.Artifact.getRunProgress = texts.GetRunProgress
	s.Artifact.getTranscript = texts.GetTranscript
	s.Artifact.getCopyLocation = texts.GetCopyLocation
	s.Artifact.deleteCopy = texts.RemoveCopy
	s.Artifact.getTranscriptCorrectionState = texts.GetTranscriptCorrectionState
	s.Artifact.getRecognitionCorrectionState = texts.GetRecognitionCorrectionState

	s.Transcript.getTranscript = texts.GetTranscript
	s.Transcript.writeTranscript = texts.WriteTranscript
	s.Article.getTranscript = texts.GetTranscript

	s.Recording.getRawTranscript = texts.GetRawTranscript
	s.Recording.getCopyLocation = texts.GetCopyLocation
}

// readImports binds what fetches the text at an address a link note points at.
func (s *Services) readImports(fetching *source.ImportURL) {
	s.files.imports = fetching
	s.Artifact.imports = fetching
}

// noteHandler answers what a vault's notes hold and what changes them.
type noteHandler struct {
	state *State

	getOpeningNote *note.GetOpeningNote
	getTypes       *note.GetTypes
	getHeadings    *note.GetHeadings
	resolveLinks   *note.ResolveLinks

	read              *note.Read
	write             *note.Write
	create            *note.Create
	editLinks         *note.EditLinks
	replaceLinks      *note.ReplaceLinks
	rename            *note.Rename
	showNeighbourhood *note.ShowNeighbourhood
	getLinkAddress    *note.GetLinkAddress
}

// searchHandler answers what the vault holds that matches what was typed.
type searchHandler struct {
	state *State
	finds *search.Search

	getNameMatches *search.GetNameMatches
	getTitles      note.GetTitles
	getTypes       *note.GetTypes
}

// artifactFiles finds the file a question about an artifact is about, in the
type artifactFiles struct {
	state          *State
	getFingerprint *file.GetFingerprint
	getNoteAddress source.GetNoteAddress
	imports        *source.ImportURL
}

// SetFolderDialog puts the machine's folder dialog in front of the person when a
// client asks for one.
func (s *Services) SetFolderDialog(dialog port.FolderDialog) {
	choose := vaults.NewChooseFolder(dialog)
	s.Vaults.chooseFolder = &choose
}

// GetPickFolder is the use case that lets an agent put the same dialog in front
// of the person, or nil where the window has none.
func (s *Services) GetPickFolder() *vaults.PickFolder {
	if s.Vaults.chooseFolder == nil {
		return nil
	}
	pick := s.Vaults.chooseFolder.GetPickFolder()
	return &pick
}
