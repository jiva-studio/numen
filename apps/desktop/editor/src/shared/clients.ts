/**
 * Every client the window talks to the application through. The transport is
 * named here.
 */
import { createClient } from '@connectrpc/connect'
import {
  AgentService,
  CommandService,
  FileService,
  NoteService,
  SearchService,
  SettingsService,
  TextVersionService,
  TextService,
  ThemeService,
  VaultService,
  VaultsService,
  WindowService,
  WorkspaceService,
} from '@numen/protocol'
import { transport } from '@numen/wire'

export const vault = createClient(VaultService, transport)

/** Whether an agent can be reached, which is the installation's and not a vault's. */
export const agentService = createClient(AgentService, transport)

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

/** The themes the window is drawn in, and how large. */
export const theme = createClient(ThemeService, transport)

/** The menu a file's text versions make, and running one of its entries. */
export const textVersions = createClient(TextVersionService, transport)

/** The live text of a file and a person's edits of it, for the views of plugins. */
export const texts = createClient(TextService, transport)

/** The commands the enabled plugins declare, and running one in the plugin's backend. */
export const pluginCommands = createClient(CommandService, transport)
