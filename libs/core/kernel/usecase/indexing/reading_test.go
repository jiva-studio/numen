package indexing

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// reading reads a vault into the index of its own that a test holds.
type reading struct {
	ReadWholeVault OpenReading
	readers        vaultport.VaultReaders
	newVault       func(tb testing.TB, notes map[string]string) domain.Vault
}

func (r reading) GetVaultReaders() vaultport.VaultReaders { return r.readers }

// openFunc opens an index in a folder of the test and returns what reads into it.
type openFunc func(t *testing.T) (reading, *WorkerIndex)

// ReadingProvider opens, in a folder of the test, an index and what reads a vault into it.
type ReadingProvider func(t *testing.T) (OpenReading, vaultport.VaultReaders, *WorkerIndex)

// RunWorkerTests runs the tests of the worker that read vaults into an index. The composition root
// imports this package, so a test of the external package gives it the provider and the function that writes a vault.
func RunWorkerTests(t *testing.T, provide ReadingProvider, newVault func(tb testing.TB, notes map[string]string) domain.Vault) {
	t.Helper()
	open := func(t *testing.T) (reading, *WorkerIndex) {
		t.Helper()
		read, readers, db := provide(t)
		return reading{ReadWholeVault: read, readers: readers, newVault: newVault}, db
	}
	t.Run("APassThatCouldNotEmbedStaysInTheList", func(t *testing.T) { testAPassThatCouldNotEmbedStaysInTheList(t, open) })
	t.Run("APassThatEmbeddedPreservesTheList", func(t *testing.T) { testAPassThatEmbeddedPreservesTheList(t, open) })
	t.Run("IndexingNamesTheSourceItIsOn", func(t *testing.T) { testIndexingNamesTheSourceItIsOn(t, open) })
	t.Run("BooksThatCouldNotBeReadStayInTheList", func(t *testing.T) { testBooksThatCouldNotBeReadStayInTheList(t, open) })
	t.Run("IndexingIsNeverAWordWithNothingBelowIt", func(t *testing.T) { testIndexingIsNeverAWordWithNothingBelowIt(t, open) })
	t.Run("ARecognitionStopsTheVectorPass", func(t *testing.T) { testARecognitionStopsTheVectorPass(t, open) })
	t.Run("AChangeOfSourcesStopsTheVectorPass", func(t *testing.T) { testAChangeOfSourcesStopsTheVectorPass(t, open) })
	t.Run("ANoteSaveStopsTheVectorPass", func(t *testing.T) { testANoteSaveStopsTheVectorPass(t, open) })
	t.Run("VersionChange_EmbedsPendingVaultContent", func(t *testing.T) { testVersionChangeEmbedsPendingVaultContent(t, open) })
	t.Run("VersionChangeOfANoteKeepsItsChunks", func(t *testing.T) { testVersionChangeOfANoteKeepsItsChunks(t, open) })
	t.Run("CancelingTheVectorTaskEndsTheEmbedCall", func(t *testing.T) { testCancelingTheVectorTaskEndsTheEmbedCall(t, open) })
	t.Run("CanceledNoteIsSkippedUntilItChanges", func(t *testing.T) { testCanceledNoteIsSkippedUntilItChanges(t, open) })
	t.Run("ProgressOfAPassAfterACancelReachesItsTotal", func(t *testing.T) { testProgressOfAPassAfterACancelReachesItsTotal(t, open) })
	t.Run("SearchSourceReadyExcludesTheCanceledNote", func(t *testing.T) { testSearchSourceReadyExcludesTheCanceledNote(t, open) })
	t.Run("APassThatEmbeddedEverythingTellsThatTheVectorsOfItsVaultAreReady", func(t *testing.T) { testAPassThatEmbeddedEverythingTellsThatTheVectorsOfItsVaultAreReady(t, open) })
	t.Run("APassWithNothingPendingTellsAgain", func(t *testing.T) { testAPassWithNothingPendingTellsAgain(t, open) })
	t.Run("APassThatCouldNotEmbedTellsNothing", func(t *testing.T) { testAPassThatCouldNotEmbedTellsNothing(t, open) })
	t.Run("APassThatWasStoppedTellsNothing", func(t *testing.T) { testAPassThatWasStoppedTellsNothing(t, open) })
	t.Run("AWorkerWithoutAListenerEmbedsAsBefore", func(t *testing.T) { testAWorkerWithoutAListenerEmbedsAsBefore(t, open) })
}
