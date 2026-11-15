import type { Result } from '@numen/plugin-sdk'

interface LinkAddress {
  readonly url: string
  readonly embed: string
}

export interface NoteContents {
  readonly body: string
  readonly link?: LinkAddress
  readonly at?: string
}

export type NoteResult = Result<NoteContents, string>
