<script setup lang="ts">
/**
 * The items held by a plugin setting of type list, one row each with a remove action, and an input
 * that adds one. Text that is blank or already listed adds nothing; the input keeps it.
 */
import { computed, ref } from 'vue'
import { Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  items: readonly string[]
  labelledBy: string
  describedBy?: string | undefined
  words: {
    readonly listAdd: string
    readonly listNew: string
    readonly listRemove: (item: string) => string
    readonly listItemTwice: (item: string) => string
  }
}>()

const emit = defineEmits<{
  add: [item: string]
  remove: [item: string]
}>()

/* --------------------------------- State ---------------------------------- */
const text = ref('')
const listedTwice = ref('')
const item = computed(() => text.value.trim())

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  text.value = (event.target as HTMLInputElement).value
  listedTwice.value = ''
}

function onAdd(): void {
  if (item.value === '') return
  if (props.items.includes(item.value)) {
    listedTwice.value = item.value
    return
  }
  emit('add', item.value)
  text.value = ''
}

function onClear(): void {
  text.value = ''
  listedTwice.value = ''
}
</script>

<template>
  <span
    class="settings__list"
    role="group"
    :aria-labelledby="labelledBy"
    :aria-describedby="describedBy"
  >
    <span v-for="one in items" :key="one" class="settings__list-item">
      <span class="settings__list-label" :title="one">{{ one }}</span>
      <Button
        variant="outline"
        size="icon-small"
        :aria-label="words.listRemove(one)"
        :title="words.listRemove(one)"
        @click="emit('remove', one)"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </span>
    <span class="settings__list-new">
      <input
        type="text"
        :value="text"
        :placeholder="words.listNew"
        :aria-labelledby="labelledBy"
        class="settings__list-input"
        @input="onInput"
        @keydown.enter.prevent="onAdd"
        @keydown.escape.prevent="onClear"
      />
      <Button variant="outline" size="small" :disabled="item === ''" @click="onAdd">
        {{ words.listAdd }}
      </Button>
    </span>
    <span v-if="listedTwice !== ''" class="settings__list-problem" role="alert">
      {{ words.listItemTwice(listedTwice) }}
    </span>
  </span>
</template>

<style scoped>
.settings__list {
  --gap: 0.375rem;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: var(--gap);
  inline-size: var(--settings-choice);
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__list-item,
.settings__list-new {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near);
  min-inline-size: 0;
}

.settings__list-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-inline-size: 0;
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}

.settings__list-input {
  --pad: 0.25rem 0.625rem;
  box-sizing: border-box;
  flex: 1 1 auto;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: var(--pad);
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.settings__list-input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.settings__list-problem {
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
