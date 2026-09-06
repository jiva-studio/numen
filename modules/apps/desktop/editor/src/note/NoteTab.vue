<script setup lang="ts">
/**
 * A note tab: the text, and the questions the file puts to the person.
 *
 * A note whose file moved past what was read stops saving and asks which of
 * the two is theirs. A note whose file is gone keeps what is on screen and
 * offers to make it again.
 */
import { watch } from 'vue'
import { Editor } from '@numen/ui'
import FileConflictPrompt from '../saving/FileConflictPrompt.vue'
import Pointing from './Pointing.vue'
import { conflictIn } from '../saving/flushing'
import { WORDS as words } from './words'
import type { NoteTabState } from './kind'

const props = defineProps<{ state: NoteTabState }>()

// The prose of a note arrives after the tab it is drawn in. The editor takes
// the keyboard it is owed once there are lines for a caret to stand on.
watch(
  () => props.state.shown.value.body,
  (body, was) => {
    if (!was && body) props.state.measure()
  },
)
</script>

<template>
  <div class="note">
    <Pointing v-if="props.state.points.value" :points="props.state.points.value" :words="words" />

    <FileConflictPrompt
      :saying="props.state.saying.value"
      :conflict="conflictIn(props.state.shown.value.state)"
      :words="words"
      @keep="props.state.keep()"
      @take="props.state.take()"
    />

    <Editor
      :ref="(editor: unknown) => props.state.drew(editor)"
      :model-value="props.state.shown.value.body"
      :change="props.state.change.value"
      class="note__text"
      @update:model-value="(body: string) => props.state.typed(body)"
      @save="props.state.save()"
      @open="(address: string) => props.state.follows(address)"
    />
  </div>
</template>

<style scoped>
/* A note fills the pane it is in: the editor scrolls, and the line it says
   something is wrong on stays where it is. */
.note {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.note__text {
  flex: 1;
  min-block-size: 0;
}
</style>
