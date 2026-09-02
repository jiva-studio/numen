<script setup lang="ts">
/**
 * One setting that is an object or a list, opened in the editor.
 *
 * No control says what such a setting holds, so the row carries a pencil and
 * the value is typed out under it. It is written as JSON5: a person may leave
 * comments and a comma after the last member, and what is written back into the
 * file is JSON.
 */
import { ref } from 'vue'
import { PenLine } from '@lucide/vue'
import { Button, Editor } from '@numen/ui'
import { read, write } from './json5'
import { WORDS as words } from './words'

const props = defineProps<{
  /** What the row is called, and under it what it means. */
  name: string
  detail: string
  /** What stands at the setting now. */
  value: unknown
}>()

const raises = defineEmits<{
  /** The value typed out, read. */
  keeps: [value: unknown]
}>()

/** Whether the editor is open, what is typed in it, and what is wrong with it. */
const open = ref(false)
const typed = ref('')
const wrong = ref('')

const opens = (): void => {
  typed.value = write(props.value ?? null)
  wrong.value = ''
  open.value = true
}

const closes = (): void => {
  open.value = false
  wrong.value = ''
}

/** What was typed, read. What is not JSON5 is said, and nothing is written. */
const keeps = (): void => {
  let value: unknown
  try {
    value = read(typed.value)
  } catch (thrown) {
    wrong.value = thrown instanceof Error ? thrown.message : `${thrown}`
    return
  }
  raises('keeps', value)
  closes()
}
</script>

<template>
  <div class="editable__row">
    <span class="editable__said">
      <span class="editable__name">{{ name }}</span>
      <span class="editable__detail">{{ detail }}</span>
    </span>
    <span class="editable__value">
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="`${words.edit} ${name}`"
        :aria-expanded="open"
        @click="open ? closes() : opens()"
      >
        <PenLine class="editable__pencil" />
      </Button>
    </span>
  </div>

  <div v-if="open" class="editable__editing">
    <p class="editable__detail">{{ words.editing }}</p>
    <Editor v-model="typed" :live="false" class="editable__editor" @save="keeps" />
    <p v-if="wrong" class="editable__wrong">{{ words.unreadable }} {{ wrong }}</p>
    <span class="editable__doing">
      <Button variant="outline" size="small" @click="closes">{{ words.cancel }}</Button>
      <Button size="small" @click="keeps">{{ words.keep }}</Button>
    </span>
  </div>
</template>

<style scoped>
/* The row reads as the rows around it, which the page lays out. */
.editable__row {
  display: grid;
  grid-template-columns: 1fr max-content;
  align-items: center;
  gap: 0 var(--numen-panel-gap);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

.editable__said {
  display: flex;
  flex-direction: column;
  gap: var(--settings-said-gap);
  min-inline-size: 0;
}

.editable__value {
  display: flex;
  align-items: center;
  justify-content: end;
}

.editable__detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.editable__pencil {
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.75;
}

/* The value typed out, under the row it belongs to. */
.editable__editing {
  display: flex;
  flex-direction: column;
  gap: var(--settings-near);
  padding-block: var(--settings-row-air);
  border-block-end: var(--numen-stroke) solid var(--numen-node-border);
}

.editable__editor {
  max-block-size: 20rem;
  overflow: auto;
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-field-bg);
}

.editable__wrong {
  margin: 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}

.editable__doing {
  display: flex;
  gap: var(--numen-panel-gap);
  justify-content: end;
}
</style>
