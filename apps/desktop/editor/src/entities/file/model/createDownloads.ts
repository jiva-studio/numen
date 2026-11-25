import { reactive } from 'vue'
import type { Result } from '@numen/result'

/** The bytes of a download brought so far, of a size the core knows. */
export interface DownloadBytes {
  readonly bytesDone: number
  readonly bytesTotal: number
}

interface DownloadAnswer extends DownloadBytes {
  readonly isKnown: boolean
}

/** The downloads of files the window runs: which paths are downloading, and the handlers it calls when that changes. */
export interface DownloadRegistry {
  fetch: (path: string) => Promise<void>
  isRunning(path: string): boolean
  getProgress: (path: string) => DownloadBytes | undefined
  /** Calls the handler with the path of each download that starts, ends or moves on; returns what stops it. */
  watch(handler: (path: string) => void): () => void
}

/** What polling the progress uses of the browser: a repeated task, and what stops it. */
export interface DownloadPorts {
  readonly every: (task: () => void, ms: number) => () => void
}

const PROGRESS_INTERVAL_MS = 500

const browserDownloadPorts: DownloadPorts = {
  every: (task, ms) => {
    const timer = setInterval(task, ms)
    return () => clearInterval(timer)
  },
}

/** What the downloads call on the core: bring a file onto the device, and ask how far that has come. */
export interface DownloadCalls {
  readonly fetchFile: (path: string) => Promise<void>
  readonly getDownloadProgress: (path: string) => Promise<Result<DownloadAnswer>>
}

export function createDownloads(
  { fetchFile, getDownloadProgress: readProgress }: DownloadCalls,
  ports: DownloadPorts = browserDownloadPorts,
): DownloadRegistry {
  const running = new Map<string, Promise<void>>()
  const progress = reactive(new Map<string, DownloadBytes>())
  const handlers = new Set<(path: string) => void>()
  const tell = (path: string) => handlers.forEach((handler) => handler(path))

  const poll = (path: string): (() => void) => {
    let isReading = false
    let isEnded = false
    const read = async (): Promise<void> => {
      if (isReading) return
      isReading = true
      try {
        const answer = await readProgress(path)
        if (isEnded || !answer.ok) return
        const { isKnown, bytesDone, bytesTotal } = answer.value
        if (!isKnown || bytesTotal <= 0) return
        const known = progress.get(path)
        if (known?.bytesDone === bytesDone && known.bytesTotal === bytesTotal) return
        progress.set(path, { bytesDone, bytesTotal })
        tell(path)
      } finally {
        isReading = false
      }
    }
    // A failed read leaves the last progress shown; the download reports its own failure.
    const stop = ports.every(() => {
      read().catch(() => undefined)
    }, PROGRESS_INTERVAL_MS)
    return () => {
      isEnded = true
      stop()
      progress.delete(path)
    }
  }

  return {
    fetch: (path) => {
      const joined = running.get(path)
      if (joined) return joined
      const stopPolling = poll(path)
      const download = fetchFile(path).finally(() => {
        stopPolling()
        running.delete(path)
        tell(path)
      })
      running.set(path, download)
      tell(path)
      return download
    },
    isRunning: (path) => running.has(path),
    getProgress: (path) => progress.get(path),
    watch: (handler) => {
      handlers.add(handler)
      return () => handlers.delete(handler)
    },
  }
}
