import { describe, expect, it } from 'vitest'
import {
  findPlatformFile,
  formatReleaseDate,
  isNewerRelease,
  parseBuildNumber,
  resolveCurrentPlatform,
} from './release'
import type { ReleaseFile } from '../model/types'

describe('release helpers', () => {
  describe('isNewerRelease', () => {
    it('returns true when remote build is strictly greater than local build', () => {
      expect(isNewerRelease(100, 105)).toBe(true)
    })

    it('returns false when remote build is less than or equal to local build', () => {
      expect(isNewerRelease(100, 100)).toBe(false)
      expect(isNewerRelease(105, 100)).toBe(false)
    })

    it('returns true when running an unversioned dev build (build 0) and remote build is positive', () => {
      expect(isNewerRelease(0, 100)).toBe(true)
    })

    it('returns false when local build is negative or remote build is non-positive', () => {
      expect(isNewerRelease(-1, 100)).toBe(false)
      expect(isNewerRelease(100, 0)).toBe(false)
      expect(isNewerRelease(100, -1)).toBe(false)
    })
  })

  describe('parseBuildNumber', () => {
    it('parses numeric values', () => {
      expect(parseBuildNumber(920)).toBe(920)
      expect(parseBuildNumber(0)).toBe(0)
      expect(parseBuildNumber(-1)).toBe(0)
    })

    it('parses pure numeric strings', () => {
      expect(parseBuildNumber('920')).toBe(920)
      expect(parseBuildNumber('  45 ')).toBe(45)
    })

    it('parses build number from beta stamps', () => {
      expect(parseBuildNumber('2026.10.1-beta.920')).toBe(920)
      expect(parseBuildNumber('0.4.0-alpha.1-364-283d1d5')).toBe(364)
      expect(parseBuildNumber('2026.10.2-144-ab260c4')).toBe(144)
    })

    it('returns zero for pure calendar or semver version without build stamp', () => {
      expect(parseBuildNumber('2026.10.2')).toBe(0)
      expect(parseBuildNumber('0.0.0-dev')).toBe(0)
    })

    it('returns zero on unparseable inputs', () => {
      expect(parseBuildNumber('invalid')).toBe(0)
      expect(parseBuildNumber(undefined)).toBe(0)
      expect(parseBuildNumber('')).toBe(0)
    })
  })

  describe('resolveCurrentPlatform', () => {
    it('identifies macOS from userAgent', () => {
      expect(
        resolveCurrentPlatform({ userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)' }),
      ).toBe('darwin')
    })

    it('identifies Windows from userAgent', () => {
      expect(
        resolveCurrentPlatform({ userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)' }),
      ).toBe('windows')
    })

    it('defaults to linux when userAgent is Linux or unknown', () => {
      expect(resolveCurrentPlatform({ userAgent: 'Mozilla/5.0 (X11; Linux x86_64)' })).toBe('linux')
      expect(resolveCurrentPlatform({ userAgent: 'unknown' })).toBe('linux')
    })
  })

  describe('findPlatformFile', () => {
    const files: readonly ReleaseFile[] = [
      {
        name: 'numen-macos.dmg',
        platform: 'macos',
        arch: 'universal',
        kind: 'dmg',
        url: 'https://dl.numen.md/numen-macos.dmg',
        size: 100,
        sha256: 'hash1',
      },
      {
        name: 'numen-windows-setup.exe',
        platform: 'windows',
        arch: 'amd64',
        kind: 'exe',
        url: 'https://dl.numen.md/numen-windows-setup.exe',
        size: 100,
        sha256: 'hash2',
      },
      {
        name: 'numen-linux-amd64.deb',
        platform: 'linux',
        arch: 'amd64',
        kind: 'deb',
        url: 'https://dl.numen.md/numen-linux-amd64.deb',
        size: 100,
        sha256: 'hash3',
      },
    ]

    it('finds darwin dmg artifact even when manifest labels platform macos', () => {
      const match = findPlatformFile(files, 'darwin')
      expect(match?.name).toBe('numen-macos.dmg')
    })

    it('finds windows setup exe artifact', () => {
      const match = findPlatformFile(files, 'windows')
      expect(match?.name).toBe('numen-windows-setup.exe')
    })

    it('finds linux deb artifact', () => {
      const match = findPlatformFile(files, 'linux')
      expect(match?.name).toBe('numen-linux-amd64.deb')
    })

    it('returns undefined when files array is empty or undefined', () => {
      expect(findPlatformFile([], 'windows')).toBeUndefined()
      expect(findPlatformFile(undefined, 'windows')).toBeUndefined()
    })
  })

  describe('formatReleaseDate', () => {
    it('formats valid ISO dates', () => {
      const formatted = formatReleaseDate('2026-10-15T12:00:00Z', 'en-GB')
      expect(formatted).toBe('15 October 2026')
    })

    it('returns empty string for invalid date', () => {
      expect(formatReleaseDate('invalid-date')).toBe('')
      expect(formatReleaseDate(undefined)).toBe('')
    })
  })
})
