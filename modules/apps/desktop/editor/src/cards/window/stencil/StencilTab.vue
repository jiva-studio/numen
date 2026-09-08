<script setup lang="ts">
/**
 * A stencil tab: the fields it names, the faces that show them, and the
 * questions the file puts.
 *
 * What is wrong with the faces and the fields is handed to the editor, which
 * stands a face's mark under that face's name and a field's under that field's
 * row. What stands against neither is said above the editor.
 */
import { computed } from 'vue'
import { StencilEditor } from '@numen/ui'
import type { InsertionPoint, Half } from '@numen/ui'
import FileConflictPrompt from '../../../shared/saving/FileConflictPrompt.vue'
import { conflictIn } from '../../../shared/saving/flushing'
import type { StencilTabState } from './stencilTabs'
import { WORDS as words } from '../words'

const props = defineProps<{ state: StencilTabState }>()

const stencil = computed(() => props.state.stencil.value)
const marks = computed(() => props.state.marks.value)
</script>

<template>
  <div class="stencil-tab">
    <FileConflictPrompt
      :saying="props.state.saying.value"
      :conflict="conflictIn(props.state.shown.value.state)"
      :words="words"
      @keep="props.state.keep()"
      @take="props.state.take()"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <StencilEditor
      class="stencil-tab__body"
      :fields="stencil.fields"
      :faces="stencil.faces"
      :wrong="marks"
      :name="words.stencil"
      @add-field="(name: string) => props.state.addsField(name)"
      @rename-field="(field: string, name: string) => props.state.namesField(field, name)"
      @remove-field="(field: string) => props.state.removesField(field)"
      @move-field="(field: string, at: InsertionPoint) => props.state.movesField(field, at)"
      @add-face="(name: string) => props.state.addsFace(name)"
      @rename-face="(id: string, name: string) => props.state.namesFace(id, name)"
      @remove-face="(id: string) => props.state.removesFace(id)"
      @move-face="(id: string, at: InsertionPoint) => props.state.movesFace(id, at)"
      @write="(id: string, half: Half, text: string) => props.state.writes(id, half, text)"
    />
  </div>
</template>

<style scoped>
/* The editor takes what the rows above it leave, and scrolls inside itself. */
.stencil-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.stencil-tab__body {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
