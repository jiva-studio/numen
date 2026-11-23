/** The log of the parts the fake sync server received and sent, read from GET /admin/part-log. */

export interface PartLogEntry {
  readonly blob: string
  readonly part: number
  readonly direction: 'put' | 'get-range'
  readonly from: number
  readonly bytes: number
}

/** The entries that send bytes twice: a part put twice, or ranges of one blob read that overlap. */
export const findRepeatedParts = (entries: readonly PartLogEntry[]): PartLogEntry[] => {
  throw new Error(`not implemented: ${entries.length} entries`)
}

/** The log of the server as the entries it holds; a log that is not a list of entries is an error. */
export const parsePartLog = (text: string): PartLogEntry[] => {
  throw new Error(`not implemented: ${text.length} characters`)
}
