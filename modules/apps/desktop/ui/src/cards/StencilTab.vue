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
import { StencilView } from '@numen/ui'
import type { CardLanding, Half } from '@numen/ui'
import Answering from '../Answering.vue'
import type { Held } from './stencil'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const sheet = computed(() => props.held.sheet())
const marks = computed(() => props.held.marks())
</script>

<template>
  <div class="stencil-tab">
    <Answering
      :saying="props.held.saying.value"
      :state="props.held.shown.value.state"
      :words="words"
      @keep="props.held.keep()"
      @take="props.held.take()"
    />

    <ul v-if="marks.whole.length" class="caution wrong" :aria-label="words.problems">
      <li v-for="(text, at) in marks.whole" :key="at">{{ text }}</li>
    </ul>

    <StencilView
      class="stencil-tab__sheet"
      :fields="sheet.fields"
      :faces="sheet.faces"
      :wrong="marks"
      :name="words.stencil"
      @add-field="(name: string) => props.held.addsField(name)"
      @rename-field="(field: string, name: string) => props.held.namesField(field, name)"
      @remove-field="(field: string) => props.held.removesField(field)"
      @move-field="(field: string, at: CardLanding) => props.held.movesField(field, at)"
      @add-face="(name: string) => props.held.addsFace(name)"
      @rename-face="(id: string, name: string) => props.held.namesFace(id, name)"
      @remove-face="(id: string) => props.held.removesFace(id)"
      @move-face="(id: string, at: CardLanding) => props.held.movesFace(id, at)"
      @write="(id: string, half: Half, text: string) => props.held.writes(id, half, text)"
    />
  </div>
</template>

<style scoped>
/* The editor takes what the bands above it leave, and scrolls inside itself. */
.stencil-tab {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
}

.stencil-tab__sheet {
  flex: 1;
  min-block-size: 0;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
