import type { NotePort } from './notes'
import type { FilePort } from './files'
import type { VaultPort } from './vault'

export interface Core extends NotePort, FilePort, VaultPort {}
