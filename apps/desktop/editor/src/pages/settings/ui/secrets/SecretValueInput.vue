<script setup lang="ts">
/**
 * A masked field a value is entered into. The value is sent once, on Enter, on leaving the field
 * or on a change, and the field is cleared as it is sent: nothing here keeps a value or shows one.
 * Whether a value is stored shows in the placeholder and in a description a screen reader reads.
 * With submit, the field keeps the value until submit resolves true, sends no second value while
 * one is in flight, and sends a value that failed again only on Enter; a failed value is marked
 * invalid with a line saying so.
 */
import { computed, ref, useId, useTemplateRef } from 'vue'
import { WORDS as words } from '../../words'
import SecretInput from './SecretInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  label: string
  hasValue: boolean
  submit?: ((value: string) => Promise<boolean>) | undefined
}>()

const emit = defineEmits<{
  (e: 'commit', value: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const entry = ref('')
const isSending = ref(false)
const failedValue = ref<string | null>(null)
const input = useTemplateRef<InstanceType<typeof SecretInput>>('input')
const stateId = useId()
const failureId = useId()
const hasFailed = computed(() => failedValue.value !== null)
const describedBy = computed(() => (hasFailed.value ? `${stateId} ${failureId}` : stateId))

defineExpose({ focus: () => input.value?.focus() })

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
  failedValue.value = null
  return onCommit()
}

function onInput(): void {
  failedValue.value = null
}

/* -------------------------------- Helpers --------------------------------- */
async function submitValue(
  submit: (value: string) => Promise<boolean>,
  value: string,
): Promise<void> {
  if (isSending.value || value === failedValue.value) return
  isSending.value = true
  try {
    if (await submit(value)) entry.value = ''
    else failedValue.value = value
  } finally {
    isSending.value = false
  }
}
</script>

<template>
  <div class="secret-value">
    <SecretInput
      ref="input"
      v-model="entry"
      type="password"
      autocomplete="off"
      spellcheck="false"
      :aria-label="label"
      :aria-describedby="describedBy"
      :aria-invalid="hasFailed"
      :placeholder="hasValue ? words.secretMasked : words.keyPlaceholder"
      @blur="onCommit"
      @change="onCommit"
      @input="onInput"
      @keydown.enter="onCommitAgain"
    />
    <span :id="stateId" class="sr-only">
      {{ hasValue ? words.secretValueSet : words.secretValueEmpty }}
    </span>
    <p v-if="hasFailed" :id="failureId" role="alert" class="secret-value__failure">
      {{ words.secretNotSaved }}
    </p>
  </div>
</template>

<style scoped>
.secret-value {
  min-inline-size: 0;
}

.secret-value__failure {
  margin: 0.25rem 0 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
