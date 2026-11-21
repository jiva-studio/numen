import type { Api, CommandRegistration, PickerHost, PickerItem } from '@numen/plugin-sdk'
import type { SetTagsRequest, SetTagsResult, TagList, TagsByPath } from '../../types'
import { getCreateTagTitle, TAG_PLACEHOLDER, TAGS_TITLE } from '../../words'
import { getPickerState } from '../lib/pickerState'
import { getTagKey } from '../../lib/tagKey'
import { getTagColor } from '../../lib/color'

/** What the command Set tags reads and writes through. */
export interface SetTagsDeps {
  readonly api: Api
  readonly picker: PickerHost
  readonly getTags: (path: string) => readonly string[]
  readonly setTags?: (changed: TagsByPath) => void
  readonly beginWrite?: () => (changed?: TagsByPath) => void
}

/**
 * The command Set tags: it opens the picker with every tag of the vault, shows the tags of the selected
 * notes as set, mixed or unset, and writes each toggle and each new tag to the notes at once. It resolves
 * when the person closes the picker.
 */
export function createSetTags(deps: SetTagsDeps): CommandRegistration {
  return {
    id: 'tags.set',
    handler: async (note) => {
      const paths = [note.path, ...note.others]
      const getColor = (tag: string): string => `var(--numen-tag-${getTagColor(tag)})`
      const listed = await deps.api.call<Record<string, never>, TagList>('GetTags', {})
      if (!listed.ok) throw new Error(listed.error.message)

      let nextWrite = 0
      let appliedWrite = 0

      const toggle = async (tag: string): Promise<readonly (readonly string[])[]> => {
        const writeId = ++nextWrite
        const finishWrite = deps.beginWrite?.()
        const written = await deps.api.call<SetTagsRequest, SetTagsResult>('SetTags', {
          paths,
          toggles: [tag],
        })
        if (!written.ok || writeId < appliedWrite) {
          finishWrite?.()
          return paths.map(deps.getTags)
        }
        appliedWrite = writeId
        const changed: Record<string, readonly string[]> = {}
        for (const one of written.value.notes) {
          if (!one.error) changed[one.path] = one.tags
        }
        if (finishWrite) finishWrite(changed)
        else deps.setTags?.(changed)
        return written.value.notes.map((one) => (one.error ? deps.getTags(one.path) : one.tags))
      }

      await deps.picker.open({
        title: TAGS_TITLE,
        placeholder: TAG_PLACEHOLDER,
        items: listed.value.tags.map(({ tag }): PickerItem => ({
          id: tag,
          title: tag,
          state: getPickerState(tag, paths.map(deps.getTags)),
          color: getColor(tag),
        })),
        recent: listed.value.recent,
        getCreateTitle: getCreateTagTitle,
        getCreateColor: getColor,
        onToggle: async (id) => ({
          id,
          title: id,
          state: getPickerState(id, await toggle(id)),
          color: getColor(id),
        }),
        onCreate: async (text) => {
          const after = await toggle(text)
          const tag = after.flat().find((one) => getTagKey(one) === getTagKey(text)) ?? text.trim()
          return { id: tag, title: tag, state: getPickerState(tag, after), color: getColor(tag) }
        },
      })
    },
  }
}
