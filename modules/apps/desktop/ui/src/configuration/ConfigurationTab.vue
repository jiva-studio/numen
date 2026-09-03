<script setup lang="ts">
/**
 * The settings file, whole, in the editor this window edits everything in.
 *
 * What is typed is written as it stands. A file the settings cannot be read out
 * of is refused, and what is wrong with it is said over the editor.
 */
import { Button, Editor } from '@numen/ui'
import type { Held } from './kind'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()
</script>

<template>
  <div class="configuration">
    <div class="configuration__head">
      <p class="configuration__where">{{ props.held.path() }}</p>
      <Button size="small" :disabled="!props.held.changed()" @click="props.held.keeps()">
        {{ words.keep }}
      </Button>
    </div>

    <p v-if="props.held.saying()" role="alert" class="configuration__wrong">
      {{ props.held.saying() }}
    </p>

    <Editor
      v-if="props.held.read()"
      :model-value="props.held.text()"
      :live="false"
      class="configuration__editor"
      :aria-label="words.called"
      @update:model-value="(said: string) => props.held.types(said)"
      @save="props.held.keeps()"
    />
    <p v-else class="configuration__waiting">{{ words.reading }}</p>
  </div>
</template>

<style scoped>
.configuration {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-node-fg);
}

/* Where the file stands, with what keeps it at the end of the line. */
.configuration__head {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap);
  padding: var(--numen-inset) var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

.configuration__where {
  min-inline-size: 0;
  margin: 0;
  color: var(--numen-hushed);
  overflow-wrap: anywhere;
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
