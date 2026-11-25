import { formatError } from '@/shared/api/errors'
import { WORDS } from '../words'

/** The kind of the tab that stands for a file while its bytes are brought. */
export const DOWNLOAD_KIND = 'download'

const GENERIC_FAILURES: ReadonlySet<string> = new Set([
  formatError(undefined),
  'internal error',
  'request timed out',
])

/** The sentence a failed download is shown as: the reason the core states, else a generic sentence. */
export const formatDownloadFailure = (error: unknown): string => {
  const text = formatError(error)
  return GENERIC_FAILURES.has(text) || text === '' ? WORDS.downloadFailed : text
}
