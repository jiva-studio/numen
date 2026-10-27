import type { Result } from '../types'
import type {
  LiveText,
  TextErrorCode,
  TextSegment,
  TextVersion,
  TextVersionId,
  VaultTexts,
} from '../texts'

// MockVaultTexts extends VaultTexts with what a test puts in it: each file's live version and its
// segments, and the edits it recorded.
export interface MockVaultTexts extends VaultTexts {
  setLive(version: TextVersion, segments: readonly TextSegment[]): void
  readonly edits: readonly { readonly path: string; readonly version: TextVersion }[]
}

// createMockVaultTexts builds the text versions of a vault in memory, as a view of the plugin
// `pluginId` works them. An edit becomes the file's live version.
export function createMockVaultTexts(pluginId = 'org.example.view'): MockVaultTexts {
  const live = new Map<string, LiveText>()
  const edits: { path: string; version: TextVersion }[] = []
  let count = 0

  return {
    edits,
    setLive(version, segments) {
      live.set(version.path, { version, segments })
    },
    async readLiveText(path) {
      return live.get(path) ?? null
    },
    async createEdit(
      path: string,
      sourceVersionId: TextVersionId,
      segments: readonly TextSegment[],
    ): Promise<Result<TextVersion, TextErrorCode>> {
      const current = live.get(path)
      if (current?.version.id !== sourceVersionId) {
        return { ok: false, error: 'missing' }
      }
      count++
      const key = `edit-${count}`
      const version: TextVersion = {
        id: `${pluginId}/${key}`,
        path,
        pluginId,
        key,
        sourceVersionId,
        kind: current.version.kind,
        file: `${key}.jsonl`,
        createTime: new Date(count),
      }
      live.set(path, { version, segments })
      edits.push({ path, version })
      return { ok: true, value: version }
    },
  }
}
