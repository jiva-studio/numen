/** One task reported by the host: the file it is on, its progress, and any error. */
export interface WatchedTask {
  readonly about: string
  readonly done?: number | undefined
  readonly total?: number | undefined
  readonly error?: string | undefined
}
