<script setup lang="ts">
/**
 * A stencil tab: the fields it names, the faces that show them, and the
 * questions the file puts.
 *
 * What is wrong with a face stands on that face, and what is wrong with a field
 * stands on that field's row.
 */
import { computed } from 'vue'
import { Stencil as StencilView } from '@numen/ui'
import type { CardLanding, Half } from '@numen/ui'
import type { Held } from './stencil'
import { standingIn } from './model'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

const sheet = computed(() => props.held.sheet())
const marks = computed(() => props.held.marks())

/** What is wrong with one face, and nothing where nothing is. */
const wrongWith = (face: string): readonly string[] => marks.value.at.get(face) ?? []

/** What is wrong with one field, and nothing where nothing is. */
const wrongWithField = (field: string): readonly string[] => marks.value.fields.get(field) ?? []

const markedFaces = computed(() =>
  sheet.value.faces.filter((face) => wrongWith(face.id).length > 0),
)

const markedFields = computed(() =>
  sheet.value.fields.filter((field) => wrongWithField(field).length > 0),
)
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

    <!-- A mark stands inside the face or the row it is about, which the editor
         draws under the identity or the name that thing carries. -->
    <Teleport
      v-for="face in markedFaces"
      :key="face.id"
      defer
      :to="standingIn('data-face-block', face.id)"
    >
      <ul class="wrong wrong--inside" :aria-label="words.wrong" data-wrong>
        <li v-for="(text, at) in wrongWith(face.id)" :key="at">{{ text }}</li>
      </ul>
    </Teleport>

    <Teleport
      v-for="field in markedFields"
      :key="field"
      defer
      :to="standingIn('data-field', field)"
    >
      <ul class="wrong wrong--inside" :aria-label="words.wrong" data-wrong>
        <li v-for="(text, at) in wrongWithField(field)" :key="at">{{ text }}</li>
      </ul>
    </Teleport>
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
  font-size: calc(var(--numen-font-size) * 12.8 / 13);
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

/* The mark stands inside a face or a row, and takes the whole width of it. */
.wrong--inside {
  flex-basis: 100%;
  color: var(--numen-alarm);
  font-size: calc(var(--numen-font-size) * 12 / 13);
  overflow-wrap: anywhere;
}
</style>
