<script setup lang="ts">
/**
 * The key of one secret a plugin declares: whether one is set, a field to type
 * one into, and the key removed. A key typed here goes to the vault and is
 * never drawn back.
 */
import { computed, ref, useId } from 'vue'
import { Button } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** What the row is about, which its name is addressed by. */
  at: string
  name: string
  isStored: boolean
  words: {
    readonly keyStored: string
    readonly keyMissing: string
    readonly keyDetail: string
    readonly keyPlaceholder: string
    readonly keySave: string
    readonly keyReplace: string
    readonly keyDelete: string
  }
}>()

const emit = defineEmits<{
  save: [key: string]
  delete: []
}>()

/* --------------------------------- State ---------------------------------- */
const typed = ref('')
const fieldId = useId()
const detail = computed(
  () =>
    `${props.isStored ? props.words.keyStored : props.words.keyMissing}. ${props.words.keyDetail}`,
)

/* -------------------------------- Handlers -------------------------------- */
function onSave() {
  const key = typed.value.trim()
  if (key === '') return
  typed.value = ''
  emit('save', key)
}

function onDelete() {
  emit('delete')
}
</script>

<template>
  <SettingRow v-slot="{ labelledBy }" :at="at" :name="name" :detail="detail">
    <span class="settings__secret">
      <label :for="fieldId" class="sr-only">{{ name }}</label>
      <input
        :id="fieldId"
        v-model="typed"
        type="password"
        autocomplete="off"
        spellcheck="false"
        :placeholder="words.keyPlaceholder"
        :aria-labelledby="labelledBy"
        class="settings__secret-input"
        @keydown.enter="onSave"
      />
      <Button variant="outline" size="small" :disabled="typed.trim() === ''" @click="onSave">
        {{ isStored ? words.keyReplace : words.keySave }}
      </Button>
      <Button v-if="isStored" variant="ghost" size="small" @click="onDelete">
        {{ words.keyDelete }}
      </Button>
    </span>
  </SettingRow>
</template>

<style scoped>
.settings__secret {
  display: flex;
  align-items: center;
  gap: var(--settings-near, 0.375rem);
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__secret-input {
  box-sizing: border-box;
  inline-size: var(--settings-choice, 18rem);
  flex: 1 1 auto;
  min-inline-size: 0;
  max-inline-size: 24rem;
  padding: 0.25rem 0.5rem;
  background-color: var(--numen-surface, transparent);
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.settings__secret-input:focus {
  outline: var(--numen-stroke, 1px) solid var(--numen-accent, currentColor);
}
</style>
