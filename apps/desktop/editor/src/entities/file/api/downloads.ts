/**
 * The calls that ask the file service about downloads: how far one has come, and whether a file waits for its
 * bytes.
 */
import { createDownloadClient } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'

export const createDownloadApi = (files: WindowClients['files']) => ({
  /** Brings the bytes of an absent file onto this device; a failure is the call's own, to be shown with `formatError`. */
  fetchFile: async (path: string): Promise<void> => {
    await files.fetchFile({ path })
  },

  getDownloadProgress: (path: string) => createDownloadClient(files).getDownloadProgress(path),

  isFileAbsent: async (path: string): Promise<boolean> => {
    const slash = path.lastIndexOf('/')
    const listed = await files.listFiles({ path: path.slice(0, Math.max(slash, 0)) })
    return listed.entries.some((one) => one.path === path && one.isAbsent)
  },
})

export type DownloadApi = ReturnType<typeof createDownloadApi>
