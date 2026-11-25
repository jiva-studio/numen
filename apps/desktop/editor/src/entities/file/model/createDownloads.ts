/** The downloads of files the window has running: which paths are being brought, and who is told when that changes. */
export interface Downloads {
  /** Brings the bytes of the file; a download of the path already running is joined, not started twice. */
  fetch: (path: string) => Promise<void>
  isRunning(path: string): boolean
  /** Calls the handler with the path of each download that starts or ends; returns what stops it. */
  watch(handler: (path: string) => void): () => void
}

export function createDownloads(fetchFile: (path: string) => Promise<void>): Downloads {
  const running = new Map<string, Promise<void>>()
  const handlers = new Set<(path: string) => void>()
  const tell = (path: string) => handlers.forEach((handler) => handler(path))

  return {
    fetch: (path) => {
      const joined = running.get(path)
      if (joined) return joined
      const download = fetchFile(path).finally(() => {
        running.delete(path)
        tell(path)
      })
      running.set(path, download)
      tell(path)
      return download
    },
    isRunning: (path) => running.has(path),
    watch: (handler) => {
      handlers.add(handler)
      return () => handlers.delete(handler)
    },
  }
}
