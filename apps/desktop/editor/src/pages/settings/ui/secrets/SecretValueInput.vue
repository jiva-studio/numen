<script setup lang="ts">
/**
 * A masked field a value is entered into. The value is sent once, on Enter, on leaving the field
 * or on a change, and the field is cleared as it is sent: nothing here keeps a value or shows one.
 * Whether a value is stored shows in the placeholder and in a description a screen reader reads.
 * With submit, the field keeps the value until submit resolves true, sends no second value while
 * one is in flight, and sends a value that failed again only on Enter; a failed value is marked
 * invalid with a line saying so.
 */
import { computed, useId, useTemplateRef } from 'vue'
import { useSecretValueEntry } from '../../model/useSecretValueEntry'
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

defineExpose({ focus: () => input.value?.focus() })

/* --------------------------------- State ---------------------------------- */
const { entry, hasFailed, actions } = useSecretValueEntry({
  getSubmit: () => props.submit,
  emitCommit: (value) => emit('commit', value),
})
const input = useTemplateRef<InstanceType<typeof SecretInput>>('input')
const stateId = useId()
const failureId = useId()
const describedBy = computed(() => (hasFailed.value ? `${stateId} ${failureId}` : stateId))

/* -------------------------------- Handlers -------------------------------- */
function onCommit(): void {
  actions.commit()
}

function onCommitAgain(): void {
  actions.commitAgain()
}

function onInput(): void {
  actions.clearFailure()
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
  --margin: 0.25rem 0 0;
  margin: var(--margin);
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
