import type { Api, CommandRegistration, PickerHost, PickerItem } from '@numen/plugin-sdk'
import type { SetTagsRequest, SetTagsResult, TagList } from '../../types'
import { getCreateTagTitle, TAG_PLACEHOLDER, TAGS_TITLE } from '../../words'
import { getPickerState, getTagKey } from '../lib/pickerState'

/** What the command Set tags reads and writes through. */
export interface SetTagsDeps {
  readonly api: Api
  readonly picker: PickerHost
  readonly getTags: (path: string) => readonly string[]
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
      const listed = await deps.api.call<Record<string, never>, TagList>('GetTags', {})
      if (!listed.ok) throw new Error(listed.error.message)

      /** The tags of each selected note after toggling one tag: the notes it could not write keep their tags. */
      const toggle = async (tag: string): Promise<readonly (readonly string[])[]> => {
        const written = await deps.api.call<SetTagsRequest, SetTagsResult>('SetTags', {
          paths,
          toggles: [tag],
        })
        if (!written.ok) return paths.map(deps.getTags)
        return written.value.notes.map((one) => (one.error ? deps.getTags(one.path) : one.tags))
      }

      await deps.picker.open({
        title: TAGS_TITLE,
        placeholder: TAG_PLACEHOLDER,
        items: listed.value.tags.map(({ tag }): PickerItem => ({
          id: tag,
          title: tag,
          state: getPickerState(tag, paths.map(deps.getTags)),
        })),
        recent: listed.value.recent,
        getCreateTitle: getCreateTagTitle,
        onToggle: async (id) => ({ id, title: id, state: getPickerState(id, await toggle(id)) }),
        onCreate: async (text) => {
          const after = await toggle(text)
          const tag = after.flat().find((one) => getTagKey(one) === getTagKey(text)) ?? text.trim()
          return { id: tag, title: tag, state: getPickerState(tag, after) }
        },
      })
    },
  }
}
