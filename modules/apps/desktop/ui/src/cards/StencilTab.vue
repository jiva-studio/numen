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
import { Stencil as StencilView } from '@numen/ui'
import type { CardLanding, Half } from '@numen/ui'
import type { Held } from './stencil'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const sheet = computed(() => props.held.sheet())
const marks = computed(() => props.held.marks())
</script>

<template>
  <div class="stencil-tab">
    <p v-if="props.held.saying()" role="alert" class="warning">{{ props.held.saying() }}</p>

    <p v-if="props.held.shown().state === 'gone'" role="status" class="warning answering">
      {{ words.gone }}
      <button type="button" class="answering__answer" @click="props.held.keep()">
        {{ words.makeAgain }}
      </button>
    </p>
    <p v-if="props.held.shown().state === 'overtaken'" role="status" class="warning answering">
      {{ words.overtaken }}
      <button type="button" class="answering__answer" @click="props.held.keep()">
        {{ words.keep }}
      </button>
      <button type="button" class="answering__answer" @click="props.held.take()">
        {{ words.take }}
      </button>
    </p>

    <ul v-if="marks.whole.length" class="warning wrong" :aria-label="words.problems">
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

/* A warning carries a filesystem path, and a long one breaks where it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

/* The question the file holds: the two answers on the line the sentence is on. */
.answering {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 0.9rem;
}

.answering__answer {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.answering__answer:focus-visible {
  outline: 1px solid currentColor;
  outline-offset: 2px;
}

.wrong {
  margin: 0;
  padding-inline-start: 1.1rem;
  list-style: disc;
}
</style>
