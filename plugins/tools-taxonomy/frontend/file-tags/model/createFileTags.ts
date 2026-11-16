import { defineComponent } from 'vue'
import type { TagMap } from '../../model/createTagMap'

export function createFileTags(_tagMap: Pick<TagMap, 'getTags'>) {
  return defineComponent({ name: 'FileTagsView', props: { path: { type: String, required: true } }, setup: () => () => null })
}
