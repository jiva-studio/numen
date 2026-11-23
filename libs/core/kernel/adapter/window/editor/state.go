// Package editor serves the vault queries and commands of the editor window.
//
// The request and response types are defined in libs/shared/protocol. The client
// connects to the handlers over HTTP in both the desktop window and a browser
// during development.
package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Session tracks the vault, tab reports and view notifications for a window.
type Session struct {
	Runtime windowsession.Runtime
	Tabs    windowsession.TabReports
	Notices *Notices
	frame   Frame
}

// Frame is the application window around the editor.
type Frame interface {
	// ShowOpenTabs tells the frame which tabs are open, so that its title follows them.
	ShowOpenTabs(open window.OpenTabs)
	// DetachTab opens a window of its own for a tab.
	DetachTab(ctx context.Context, tab window.DetachedTab) error
}

// NewSession opens the editor's vault and tab reporting state.
func NewSession(frame Frame) *Session {
	return &Session{Notices: newNotices(), frame: frame}
}

// GetCurrentVault is the vault the window shows.
func (s *Session) GetCurrentVault() domain.Vault { return s.Runtime.GetCurrentVault() }

// GetShownVault is the vault the window shows, and false while it shows none.
func (s *Session) GetShownVault() (domain.Vault, bool) {
	current := s.GetCurrentVault()
	return current, current.Path != ""
}

// GetVaultState is what the window knows about the vault it shows: whether the first scan is done and
// what stopped it.
func (s *Session) GetVaultState() window.VaultState {
	return windowsession.GetVaultState(&s.Runtime)
}

// ReportOpenTabs keeps the tabs the window reported, wakes whoever waits for a report, and tells
// the application.
func (s *Session) ReportOpenTabs(open window.OpenTabs) {
	s.Tabs.ReportOpenTabs(open)
	s.ShowOpenTabs(open)
}

// ShowOpenTabs sends the supplied tab snapshot to the current window frame.
func (s *Session) ShowOpenTabs(open window.OpenTabs) {
	if s.frame != nil {
		s.frame.ShowOpenTabs(open)
	}
}

func (s *Session) CanOpenTabWindow() bool { return s.frame != nil }

func (s *Session) OpenTabWindow(ctx context.Context, tab window.DetachedTab) error {
	return s.frame.DetachTab(ctx, tab)
}

// GetOpenTabs returns the last tab report from the window.
func (s *Session) GetOpenTabs() window.OpenTabs { return s.Tabs.GetOpenTabs() }

// GetFrameDialog returns the native dialog of the frame, and nothing when the frame has none.
func GetFrameDialog(frame Frame) PathDialog {
	dialog, _ := frame.(PathDialog)
	return dialog
}
