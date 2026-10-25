import type { Disposable, Result } from '../types'
import type {
  TextErrorCode,
  TextListing,
  TextPage,
  TextPart,
  TextVersion,
  TextVersionId,
  VaultTexts,
} from '../texts'

// MockVaultTexts extends VaultTexts with what a test puts in it: each version with its text, and
// which one is live. A page holds one part.
export interface MockVaultTexts extends VaultTexts {
  putVersion(version: TextVersion, parts: readonly TextPart[]): void
  setLive(path: string, versionId: TextVersionId): void
}

// createMockVaultTexts builds the text versions of a vault in memory.
export function createMockVaultTexts(): MockVaultTexts {
  const versions = new Map<string, { version: TextVersion; parts: readonly TextPart[] }[]>()
  const live = new Map<string, TextVersionId>()
  const handlers = new Set<(path: string) => void>()

  function notify(path: string): void {
    for (const handler of handlers) {
      handler(path)
    }
  }

  return {
    putVersion(version, parts) {
      const list = (versions.get(version.path) ?? []).filter((one) => one.version.id !== version.id)
      versions.set(version.path, [...list, { version, parts }])
      notify(version.path)
    },
    setLive(path, versionId) {
      live.set(path, versionId)
      notify(path)
    },
    async listVersions(path): Promise<TextListing> {
      return {
        versions: (versions.get(path) ?? []).map((one) => one.version),
        liveVersionId: live.get(path) ?? '',
      }
    },
    async readText(path, versionId, cursor = ''): Promise<Result<TextPage, TextErrorCode>> {
      const found = (versions.get(path) ?? []).find((one) => one.version.id === versionId)
      if (!found) {
        return { ok: false, error: 'missing' }
      }
      const from = cursor === '' ? 0 : Number(cursor)
      const next = from + 1 < found.parts.length ? String(from + 1) : ''
      return { ok: true, value: { parts: found.parts.slice(from, from + 1), nextCursor: next } }
    },
    onVersionsChange(handler): Disposable {
      handlers.add(handler)
      return { dispose: () => handlers.delete(handler) }
    },
  }
}
