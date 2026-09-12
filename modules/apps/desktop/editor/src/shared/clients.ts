/**
 * Every client the window talks to the application through. The transport is
 * named here.
 */
import { createClient } from '@connectrpc/connect'
import {
  AgentService,
  ArticleService,
  ArtifactService,
  BookService,
  CardsService,
  DocumentService,
  FileService,
  NoteService,
  OcrService,
  PresetsService,
  RecordingService,
  SearchService,
  SettingsService,
  ThemeService,
  TranscriptService,
  VaultService,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'
import { transport } from '@numen/wire'

export const vault = createClient(VaultService, transport)

/** Whether an agent can be reached, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)

/** Where the person stands in the vault: what they have open, and where they are sent. */
export const workspace = createClient(WorkspaceService, transport)

/** The tree the vault is filed in: what stands where, and moving it about. */
export const files = createClient(FileService, transport)

/** What a note holds, what it is joined to, and every way of writing one. */
export const notes = createClient(NoteService, transport)

/** What the vault holds that answers what a person typed. */
export const search = createClient(SearchService, transport)

export const vaultsService = createClient(VaultsService, transport)

/** The file this installation is configured in, which is no vault's. */
export const settingsService = createClient(SettingsService, transport)

/** This window itself, which is the editor and not the one cards are run in. */
export const windowService = createClient(WindowService, transport)

/** What a model has made from the files of the vault. */
export const artifacts = createClient(ArtifactService, transport)

/** The decks, the cards and the stencils a vault holds. */
export const cardsService = createClient(CardsService, transport)

/** The presets a review is run under. */
export const presetsService = createClient(PresetsService, transport)

/** The recordings the vault holds, by their address and their duration. */
export const recordings = createClient(RecordingService, transport)

/** The cues a recording is written down in. */
export const transcripts = createClient(TranscriptService, transport)

/** The prose a recording is written up as. */
export const articles = createClient(ArticleService, transport)

/** The themes the window is drawn in, and how large. */
export const theme = createClient(ThemeService, transport)

/** What a book holds: its spine, its parts, and the markup of one document. */
export const books = createClient(BookService, transport)

/** The pages a document is laid out in. */
export const documents = createClient(DocumentService, transport)

/** Where the text of a document stands on its pages. */
export const ocr = createClient(OcrService, transport)
