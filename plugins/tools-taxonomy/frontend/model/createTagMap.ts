import { getCurrentScope, onScopeDispose, shallowReactive, type EffectScope } from 'vue'
import type { ApiError, PluginEvents, Result } from '@numen/plugin-sdk'
import type { TagMapResponse, TagsByPath } from '../types'

interface TagMapApi {
  getTagMap(options?: { signal?: AbortSignal }): Promise<Result<TagMapResponse, ApiError>>
}

interface TagMapDeps {
  readonly api: TagMapApi
  readonly events?: Pick<PluginEvents, 'onVaultChange'>
}

/** The tags of the notes of the vault by path, read when asked and refreshed on vault changes while in use. */
export interface TagMap {
  getTags(path: string): readonly string[]
  setTags(changed: TagsByPath): void
  beginWrite(): (changed?: TagsByPath) => void
  waitForLoad(): Promise<void>
  stop(): void
}

const NO_TAGS: readonly string[] = []

const isSameTags = (left: readonly string[] | undefined, right: readonly string[]): boolean =>
  left !== undefined &&
  left.length === right.length &&
  left.every((tag, index) => tag === right[index])

const setPathTags = (
  tags: Map<string, readonly string[]>,
  path: string,
  found: readonly string[],
): void => {
  if (found.length === 0) {
    tags.delete(path)
    return
  }
  if (!isSameTags(tags.get(path), found)) tags.set(path, found)
}

const replaceTags = (tags: Map<string, readonly string[]>, whole: TagsByPath): void => {
  for (const path of tags.keys()) {
    if ((whole[path]?.length ?? 0) === 0) tags.delete(path)
  }
  for (const [path, found] of Object.entries(whole)) setPathTags(tags, path, found)
}

/**
 * The map of the tags of the vault: read from GetTagMap on the first read or waitForLoad call, updated
 * directly when SetTags writes a note, and refreshed on onVaultChange while a view or effect scope is
 * active. A caller that reads the tags of one path is notified when the tags of that path change and of
 * no other.
 */
export function createTagMap({ api, events }: TagMapDeps): TagMap {
  const tags = shallowReactive(new Map<string, readonly string[]>())
  const scopes = new Set<EffectScope>()
  let controller: AbortController | undefined
  let currentRead: Promise<void> | undefined
  let isFresh = false
  let isStopped = false
  let needsReload = false
  let version = 0

  const trackScope = (): void => {
    const scope = getCurrentScope()
    if (!scope || scopes.has(scope)) return
    scopes.add(scope)
    onScopeDispose(() => {
      scopes.delete(scope)
    })
  }

  const readOnce = async (): Promise<boolean> => {
    const current = new AbortController()
    controller = current
    const result = await api
      .getTagMap({ signal: current.signal })
      .catch((): Result<TagMapResponse, ApiError> => ({
        ok: false,
        error: { code: 'unavailable', message: '' },
      }))
    if (isStopped || current.signal.aborted || !result.ok) return false
    version += 1
    replaceTags(tags, result.value.tags)
    return true
  }

  const startRead = (): Promise<void> => {
    if (isStopped) return Promise.resolve()
    if (currentRead) return currentRead
    needsReload = false
    const pass = readOnce().then((didRead) => {
      currentRead = undefined
      isFresh = didRead && !needsReload && !isStopped
      if (needsReload && !isStopped && scopes.size > 0) requestLoad()
    })
    currentRead = pass
    return pass
  }

  function requestLoad(): void {
    startRead().catch(() => undefined)
  }

  const invalidate = (): void => {
    if (isStopped) return
    if (currentRead) {
      needsReload = true
      return
    }
    isFresh = false
    if (scopes.size > 0) requestLoad()
  }

  const applySetTags = (changed: TagsByPath): void => {
    version += 1
    for (const [path, found] of Object.entries(changed)) setPathTags(tags, path, found)
    if (!currentRead) return
    needsReload = true
    controller?.abort()
  }

  const subscription = events?.onVaultChange(invalidate)

  return {
    getTags: (path) => {
      trackScope()
      if (!isFresh) requestLoad()
      return tags.get(path) ?? NO_TAGS
    },
    setTags: applySetTags,
    beginWrite: () => {
      const startVersion = version
      let isDone = false
      return (changed) => {
        if (isDone || isStopped) return
        isDone = true
        if (!changed) return
        if (startVersion !== version) {
          invalidate()
          return
        }
        applySetTags(changed)
      }
    },
    waitForLoad: () => (isFresh ? Promise.resolve() : startRead()),
    stop: () => {
      isStopped = true
      needsReload = false
      scopes.clear()
      controller?.abort()
      subscription?.dispose()
    },
  }
}
