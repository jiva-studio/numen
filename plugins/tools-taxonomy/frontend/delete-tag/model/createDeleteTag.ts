import type { Api, CommandItem, CommandRegistration } from '@numen/plugin-sdk'
import type { SetTagsRequest, SetTagsResult, TagMapResponse, TagsByPath } from '../../types'
import { getTagKey } from '../../lib/tagKey'
import { getDeleteTagDetail } from '../../words'

/** What the command Delete tag reads and writes through. */
export interface DeleteTagDeps {
  readonly api: Api
  readonly setTags?: (changed: TagsByPath) => void
  readonly beginWrite?: () => (changed?: TagsByPath) => void
}

interface Spelled {
  readonly tag: string
  readonly paths: string[]
}

const readTagMap = async (api: Api): Promise<TagsByPath> => {
  const read = await api.call<Record<string, never>, TagMapResponse>('GetTagMap', {})
  if (!read.ok) throw new Error(read.error.message)
  return read.value.tags
}

const groupByTag = (map: TagsByPath): Map<string, Spelled> => {
  const grouped = new Map<string, Spelled>()
  for (const path of Object.keys(map).sort()) {
    const keys = new Set<string>()
    for (const tag of map[path] ?? []) {
      const key = getTagKey(tag)
      if (key === '' || keys.has(key)) continue
      keys.add(key)
      const found = grouped.get(key) ?? { tag, paths: [] }
      found.paths.push(path)
      grouped.set(key, found)
    }
  }
  return grouped
}

const removeTag = async (
  deps: DeleteTagDeps,
  tag: string,
  paths: readonly string[],
): Promise<void> => {
  const finishWrite = deps.beginWrite?.()
  const written = await deps.api.call<SetTagsRequest, SetTagsResult>('SetTags', {
    paths,
    toggles: [tag],
  })
  if (!written.ok) {
    finishWrite?.()
    throw new Error(written.error.message)
  }
  const changed: Record<string, readonly string[]> = {}
  for (const one of written.value.notes) {
    if (!one.error) changed[one.path] = one.tags
  }
  if (finishWrite) finishWrite(changed)
  else deps.setTags?.(changed)
  const failed = written.value.notes.filter((one) => one.error)
  if (failed.length > 0) {
    throw new Error(failed.map((one) => `${one.path}: ${one.error}`).join('; '))
  }
}

/**
 * The command Delete tag: it lists every tag that a note has, with the number of notes, and removes the
 * tag picked from each of those notes in one SetTags call. A note that cannot be written keeps the tag
 * and the command fails with the reason.
 */
export function createDeleteTag(deps: DeleteTagDeps): CommandRegistration {
  return {
    id: 'tags.delete',
    listItems: async (): Promise<readonly CommandItem[]> => {
      const grouped = groupByTag(await readTagMap(deps.api))
      return [...grouped.values()]
        .sort((left, right) => left.tag.localeCompare(right.tag))
        .map(({ tag, paths }) => ({
          id: tag,
          title: tag,
          detail: getDeleteTagDetail(paths.length),
        }))
    },
    handler: async (_note, itemId) => {
      if (!itemId) return
      const paths = groupByTag(await readTagMap(deps.api)).get(getTagKey(itemId))?.paths ?? []
      if (paths.length > 0) await removeTag(deps, itemId, paths)
    },
  }
}
