import { defineComponent, h } from 'vue'
import PlexTags from './PlexTags.vue'

/** Where the tags of a path come from. */
export interface TagSource {
  getTags(path: string): readonly string[]
}

/** The view of the slot plexGraph.nodeBelow: the tag chips of the path of its node. */
export function createPlexTags(tagMap: TagSource) {
  return defineComponent({
    name: 'PlexTagsView',
    props: { path: { type: String, required: true } },
    setup: (props) => () => h(PlexTags, { tags: tagMap.getTags(props.path) }),
  })
}
