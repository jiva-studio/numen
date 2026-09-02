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
import { WORDS as words } from './words'
import type { Held } from './kind'

const props = defineProps<{ held: Held }>()

// The prose of a note arrives after the tab it is drawn in. The editor takes
// the keyboard it is owed once there are lines for a caret to stand on.
watch(
  () => props.held.shown().body,
  (body, was) => {
    if (!was && body) props.held.measure()
  },
)
</script>

<template>
  <div class="note">
    <p v-if="props.held.saying()" role="alert" class="warning">{{ props.held.saying() }}</p>

    <p v-if="props.held.shown().state === 'gone'" role="status" class="warning overtaken">
      {{ words.gone }}
      <button type="button" class="overtaken__answer" @click="props.held.keep()">
        {{ words.makeAgain }}
      </button>
    </p>
    <p v-if="props.held.shown().state === 'overtaken'" role="status" class="warning overtaken">
      {{ words.overtaken }}
      <button type="button" class="overtaken__answer" @click="props.held.keep()">
        {{ words.keep }}
      </button>
      <button type="button" class="overtaken__answer" @click="props.held.take()">
        {{ words.take }}
      </button>
    </p>

    <Editor
      :ref="(editor: unknown) => props.held.drew(editor)"
      :model-value="props.held.shown().body"
      :change="props.held.change()"
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

/* A warning and a failure carry filesystem paths, and a long one breaks where
   it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: calc(var(--numen-font-size) * 12.8 / 13);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

/* The question a note holds: the band a refusal is said in, with the two
   answers on the same line as the sentence, so the band stands one line high. */
.overtaken {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0 0.9rem;
}

.overtaken__answer {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  text-underline-offset: 0.15em;
  cursor: pointer;
}

.overtaken__answer:hover {
  text-decoration-thickness: 2px;
}

.overtaken__answer:focus-visible {
  outline: 1px solid currentColor;
  outline-offset: 2px;
}
</style>
