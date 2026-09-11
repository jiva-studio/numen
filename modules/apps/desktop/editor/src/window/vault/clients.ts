/**
 * Clients for services the window talks to over the schema.
 */
import { createClient } from '@connectrpc/connect'
import {
  AgentService,
  FileService,
  NoteService,
  SearchService,
  SettingsService,
  VaultService,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'
import { transport } from '@numen/wire'

/** The window every question about a window names. */
export const WINDOW = 'editor'

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
