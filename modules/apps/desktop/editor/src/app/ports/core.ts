/**
 * Everything the window asks of the vault, and the words the vault speaks.
 *
 * Nothing here is about drawing: a kind translates these into what it holds,
 * and this is what every one of them starts from.
 */
import type { NotePort } from './notes'
import type { FilePort } from './files'
import type { SettingsPort } from './settings'
import type { VaultPort } from './vault'

export interface Core extends NotePort, FilePort, SettingsPort, VaultPort {}
