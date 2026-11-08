<script setup lang="ts">
/**
 * A masked field a value is entered into. The value is sent once, on Enter, on leaving the field
 * or on a change, and the field is cleared as it is sent: nothing here keeps a value or shows one.
 * With submit, the field keeps the value until submit resolves true, sends no second value while
 * one is in flight, and sends a value that failed again only on Enter.
 */
import { onMounted, ref, useTemplateRef } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  label: string
  placeholder: string
  submit?: ((value: string) => Promise<boolean>) | undefined
  /** Whether the field takes the focus as it appears. */
  isFocused?: boolean
}>()

const emit = defineEmits<{
  (e: 'commit', value: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const entry = ref('')
const field = useTemplateRef<HTMLInputElement>('field')
const isSending = ref(false)
/** The value whose last submit failed, until the field changes. */
let failedValue: string | null = null

onMounted(() => {
  if (props.isFocused) field.value?.focus()
})

/* -------------------------------- Handlers -------------------------------- */
async function onCommit(): Promise<void> {
  const value = entry.value.trim()
  if (value === '') return
  if (props.submit) {
    await submitValue(props.submit, value)
    return
  }
  entry.value = ''
  emit('commit', value)
}

function onCommitAgain(): Promise<void> {
  failedValue = null
  return onCommit()
}

function onInput(): void {
  failedValue = null
}

/* -------------------------------- Helpers --------------------------------- */
async function submitValue(
  submit: (value: string) => Promise<boolean>,
  value: string,
): Promise<void> {
  if (isSending.value || value === failedValue) return
  isSending.value = true
  try {
    if (await submit(value)) entry.value = ''
    else failedValue = value
  } finally {
    isSending.value = false
  }
}
</script>

<template>
  <input
    ref="field"
    v-model="entry"
    type="password"
    autocomplete="off"
    spellcheck="false"
    :aria-label="label"
    :placeholder="placeholder"
    class="secret-value"
    @blur="onCommit"
    @change="onCommit"
    @input="onInput"
    @keydown.enter="onCommitAgain"
  />
</template>

<style scoped>
.secret-value {
  box-sizing: border-box;
  inline-size: 100%;
  min-inline-size: 0;
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.secret-value:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
