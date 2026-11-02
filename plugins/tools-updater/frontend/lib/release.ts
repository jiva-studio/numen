/**
 * Words and dates the update popover shows for a release.
 */
import type { Platform } from '../types'
import { WORDS as words } from '../words'

/** Formats an ISO 8601 timestamp for display; empty where the timestamp is missing or invalid. */
export function formatReleaseDate(isoDate: string | undefined, locale: string = 'en-GB'): string {
  if (!isoDate) return ''
  const date = new Date(isoDate)
  if (Number.isNaN(date.valueOf())) return ''
  return date.toLocaleDateString(locale, { day: 'numeric', month: 'long', year: 'numeric' })
}

/** The instruction to install an update on the platform. */
export function getPlatformMessage(platform: Platform): string {
  switch (platform) {
    case 'windows':
      return words.windowsMessage
    case 'darwin':
      return words.darwinMessage
    case 'linux':
      return words.linuxMessage
  }
}
