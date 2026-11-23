<script setup lang="ts">
/**
 * The dialog that sets up a new identity in two steps: the twelve recovery words, shown once, and
 * the check of three of them. Nothing is kept unless the check passes. Escape and a press on the
 * backdrop cancel the setup, and Tab stays inside the dialog.
 */
import { computed, onMounted, useId, useTemplateRef, watch } from 'vue'
import { WORDS as words } from '../../words'
import ConfirmWords from './ConfirmWords.vue'
import IdentitySetupActions from './IdentitySetupActions.vue'
import type { IdentitySetupProps } from './IdentitySetup.types'
import RecoveryWords from './RecoveryWords.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<IdentitySetupProps>()

const emit = defineEmits<{
  (e: 'change', position: number, value: string): void
  (e: 'check'): void
  (e: 'back'): void
  (e: 'keep'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const titleId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')
const isWords = computed(() => props.stage === 'words')

/* ---------------------------------- Hooks --------------------------------- */
onMounted(focusFirst)
watch(() => props.stage, focusFirst, { flush: 'post' })

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (isWords.value) emit('check')
  else if (props.isComplete) emit('keep')
}

function onKeepFocus(event: KeyboardEvent): void {
  const items = Array.from(
    dialog.value?.querySelectorAll<HTMLElement>('input, button') ?? [],
  ).filter((one) => !one.hasAttribute('disabled'))
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) return
  if (event.shiftKey && event.target === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && event.target === last) {
    event.preventDefault()
    first.focus()
  }
}

/* -------------------------------- Helpers --------------------------------- */
function focusFirst(): void {
  dialog.value?.querySelector<HTMLElement>('input, button')?.focus()
}
</script>

<template>
  <div
    class="identity-setup__backdrop"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
    @keydown.tab="onKeepFocus"
  >
    <form
      ref="dialog"
      class="connection-dialog"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      @submit.prevent="onSubmit"
    >
      <template v-if="isWords">
        <h2 :id="titleId" class="connection-dialog__title">{{ words.identitySetupTitle }}</h2>
        <p class="connection-dialog__text">{{ words.identityWriteDown }}</p>
        <RecoveryWords :recovery-words="recoveryWords" />
        <p class="caution caution--card">{{ words.identityNobodyRecovers }}</p>
      </template>

      <template v-else>
        <h2 :id="titleId" class="connection-dialog__title">{{ words.identityCheckTitle }}</h2>
        <p class="connection-dialog__text">{{ words.identityConfirmHint(positions) }}</p>
        <ConfirmWords
          :positions="positions"
          :entered="entered"
          :is-disabled="isBusy"
          @change="(position, value) => emit('change', position, value)"
          @submit="onSubmit"
        />
        <p v-if="problem !== ''" role="alert" class="connection-dialog__problem">
          {{ problem }}
        </p>
      </template>

      <IdentitySetupActions
        :stage="stage"
        :is-busy="isBusy"
        :is-complete="isComplete"
        @cancel="emit('cancel')"
        @back="emit('back')"
      />
    </form>
  </div>
</template>

<style scoped>
.identity-setup__backdrop {
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--numen-gutter);
  background: color-mix(in oklab, var(--numen-surface), transparent 30%);
}

.connection-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  box-sizing: border-box;
  inline-size: 100%;
  max-inline-size: 28rem;
  max-block-size: 100%;
  overflow-y: auto;
  padding: 1rem 1.25rem;
  background: var(--numen-raised);
  border: var(--numen-stroke) solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  line-height: var(--numen-line-height);
}

.connection-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.connection-dialog__text,
.connection-dialog__problem {
  margin: 0;
}

.connection-dialog__problem {
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
