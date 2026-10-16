/**
 * Pure functions for version comparison, platform resolution, and release assets.
 */
import type { ReleaseFile } from '../model/types'

export type ClientPlatform = 'darwin' | 'windows' | 'linux'

/**
 * Checks whether the remote release build is newer than the local build.
 */
export function isNewerRelease(currentBuild: number, latestBuild: number): boolean {
  if (currentBuild <= 0 || latestBuild <= 0) {
    return false
  }
  return latestBuild > currentBuild
}

/**
 * Extracts a numeric build count from an integer or formatted version stamp.
 */
export function parseBuildNumber(val: string | number | undefined): number {
  if (typeof val === 'number') {
    return Number.isFinite(val) && val > 0 ? Math.floor(val) : 0
  }
  if (!val || typeof val !== 'string') {
    return 0
  }

  const match = val.match(/(?:-beta\.|\b)(\d+)(?:-[a-f0-9]{7})?$/)
  const candidate = match && match[1] ? match[1] : val.trim()
  const parsed = parseInt(candidate, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

/**
 * Identifies the client operating system from the browser environment.
 */
export function resolveCurrentPlatform(nav?: {
  userAgent?: string
  platform?: string
}): ClientPlatform {
  const agent = nav?.userAgent ?? (typeof navigator !== 'undefined' ? navigator.userAgent : '')
  const plat = nav?.platform ?? (typeof navigator !== 'undefined' ? navigator.platform : '')
  const combined = `${agent} ${plat}`.toLowerCase()

  if (combined.includes('mac') || combined.includes('darwin')) {
    return 'darwin'
  }
  if (combined.includes('win')) {
    return 'windows'
  }
  return 'linux'
}

/**
 * Picks the primary artifact for the specified operating system.
 */
export function findPlatformFile(
  files: readonly ReleaseFile[] | undefined,
  platform: ClientPlatform,
): ReleaseFile | undefined {
  if (!files || files.length === 0) {
    return undefined
  }

  if (platform === 'darwin') {
    return (
      files.find((f) => f.platform === 'darwin' && f.kind === 'dmg') ??
      files.find((f) => f.platform === 'darwin')
    )
  }

  if (platform === 'windows') {
    return (
      files.find((f) => f.platform === 'windows' && f.kind === 'exe') ??
      files.find((f) => f.platform === 'windows' && f.name.endsWith('.exe')) ??
      files.find((f) => f.platform === 'windows')
    )
  }

  return (
    files.find((f) => f.platform === 'linux' && f.name.endsWith('.deb')) ??
    files.find((f) => f.platform === 'linux' && f.kind === 'tar.gz') ??
    files.find((f) => f.platform === 'linux')
  )
}

/**
 * Formats an ISO 8601 timestamp for human display.
 */
export function formatReleaseDate(isoDate?: string, locale: string = 'en-GB'): string {
  if (!isoDate) {
    return ''
  }
  const date = new Date(isoDate)
  if (Number.isNaN(date.valueOf())) {
    return ''
  }
  return date.toLocaleDateString(locale, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}
