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
import Answering from '../Answering.vue'
import { WORDS as words } from './words'
import type { NoteTabState } from './kind'

const props = defineProps<{ held: NoteTabState }>()

// The prose of a note arrives after the tab it is drawn in. The editor takes
// the keyboard it is owed once there are lines for a caret to stand on.
watch(
  () => props.held.shown.value.body,
  (body, was) => {
    if (!was && body) props.held.measure()
  },
)
</script>

<template>
  <div class="note">
    <Answering
      :saying="props.held.saying.value"
      :state="props.held.shown.value.state"
      :words="words"
      @keep="props.held.keep()"
      @take="props.held.take()"
    />

    <Editor
      :ref="(editor: unknown) => props.held.drew(editor)"
      :model-value="props.held.shown.value.body"
      :change="props.held.change.value"
      class="note__text"
      @update:model-value="(body: string) => props.held.typed(body)"
      @save="props.held.save()"
      @open="(address: string) => props.held.follows(address)"
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
