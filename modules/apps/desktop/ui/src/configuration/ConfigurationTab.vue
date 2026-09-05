<script setup lang="ts">
/**
 * The settings file, whole, in the editor this window edits everything in.
 *
 * It is written as JSON, so it is read as JSON and set in the face code is set
 * in. What is typed is kept the way a note is kept, and a file the settings
 * cannot be read out of is refused with what is wrong with it. A file that
 * moved past what was read puts the person the two answers a note puts.
 */
import { Editor } from '@numen/ui'
import Caution from '../Caution.vue'
import type { ConfigurationTabState } from './kind'
import { WORDS as words } from './words'

const props = defineProps<{ held: ConfigurationTabState }>()
</script>

<template>
  <div class="configuration">
    <p v-if="props.held.saying()" role="alert" class="configuration__wrong">
      {{ props.held.saying() }}
    </p>

    <Caution v-if="props.held.overtaken()" role="status" answering>
      {{ words.overtaken }}
      <button type="button" class="answer" @click="props.held.keep()">{{ words.keep }}</button>
      <button type="button" class="answer" @click="props.held.take()">{{ words.take }}</button>
    </Caution>

    <Editor
      v-if="props.held.read()"
      :model-value="props.held.text()"
      :live="false"
      language="json"
      class="configuration__editor"
      :aria-label="words.called"
      @update:model-value="(said: string) => props.held.types(said)"
      @save="props.held.keeps()"
    />
    <p v-else class="configuration__waiting">{{ words.reading }}</p>
  </div>
</template>

<style scoped>
/* The file fills the pane it is in: the editor scrolls, and the line saying
   something is wrong stays where it is. */
.configuration {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

/* What is wrong with what was typed, which is the one thing to catch the eye. */
.configuration__wrong {
  flex: none;
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.configuration__editor {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}

.configuration__waiting {
  margin: 0;
  padding: var(--numen-gutter);
  color: var(--numen-hushed);
}
</style>
