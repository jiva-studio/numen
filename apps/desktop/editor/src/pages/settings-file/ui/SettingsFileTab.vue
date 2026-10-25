<script setup lang="ts">
import { Editor, Waiting } from '@numen/ui'
import type { SettingsFileTabState } from '../model/useSettingsFile'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: SettingsFileTabState }>()

/* --------------------------------- State ---------------------------------- */
const { errorMessage, isStale, isRead, text } = props.state

/* -------------------------------- Handlers -------------------------------- */
function onKeep() {
  void props.state.keep()
}

function onTake() {
  void props.state.take()
}

function onUpdateModelValue(content: string) {
  props.state.type(content)
}

function onSave() {
  void props.state.save()
}
</script>

<template>
  <div class="settings-file">
    <p v-if="errorMessage" role="alert" class="settings-file__wrong">
      {{ errorMessage }}
    </p>

    <p v-if="isStale" class="caution caution--conflict" role="status">
      {{ words.stale }}
      <button type="button" class="answer" @click="onKeep">{{ words.keep }}</button>
      <button type="button" class="answer" @click="onTake">{{ words.take }}</button>
    </p>

    <Editor
      v-if="isRead"
      :model-value="text"
      :is-live="false"
      language="json"
      class="settings-file__editor"
      :aria-label="words.called"
      @update:model-value="onUpdateModelValue"
      @save="onSave"
    />
    <Waiting v-else :label="words.loading" />
  </div>
</template>

<style scoped>
.settings-file {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.settings-file__wrong {
  flex: none;
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}

.settings-file__editor {
  flex: 1;
  min-block-size: 0;
  overflow: auto;
}
</style>
