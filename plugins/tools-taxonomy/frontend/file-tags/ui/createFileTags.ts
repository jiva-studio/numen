import { defineComponent, h } from 'vue'
import FileTags from './FileTags.vue'

/** Where the tags of a path come from. */
export interface TagSource {
  getTags(path: string): readonly string[]
}

/** The view of the slot fileTree.rowEnd: the tag dots of the path of its row. */
export function createFileTags(tagMap: TagSource) {
  return defineComponent({
    name: 'FileTagsView',
    props: { path: { type: String, required: true } },
    setup: (props) => {
      tagMap.getTags(props.path)
      return () => h(FileTags, { tags: tagMap.getTags(props.path) })
    },
  })
}
