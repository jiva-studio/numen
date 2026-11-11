package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

// Mount registers the plugin's services on s, reporting text versions to the host the environment
// names.
func Mount(s *server.Server) error {
	connected, err := hostclient.OpenFromEnv()
	if err != nil {
		return err
	}
	return mount(s, connected)
}

func mount(s *server.Server, connected *hostclient.Host) error {
	reader := books.NewReader(connected.OpenFiles)
	writer := texts.NewWriter(connected.OpenVault, connected.OpenTexts, connected.OpenStore, connected.GetQueue(), connected.ReportError)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook releases cached books"
	s.OnClose(func() error {
		reader.Close()
		return nil
	})
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server close hook stops running text extraction jobs"
	s.OnClose(func() error {
		writer.Close()
		return nil
	})
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="mount propagates any API registration failure"
	if err := mountMethods(s, reader); err != nil {
		return err
	}
	s.RegisterRoute(entriesPattern, NewEntryServer(reader, connected.ReportError))
	commands, err := newCommands(writer)
	// gomutants:disable-next-line BRANCH_IF reason="registering one static command id on a new registry cannot fail"
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. A request naming no vault works the
// vault those name, and text versions are reported to the host they name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		connected, err := hostclient.OpenFromVars(vars)
		// gomutants:disable-next-line BRANCH_IF reason="NewLinkedHandler already validates vars before invoking mount callback"
		if err != nil {
			return err
		}
		return mount(s, connected)
	})
}
