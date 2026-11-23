<script setup lang="ts">
/**
 * The dialog that links a device to the account by a QR code. On a device that has the account it shows the QR code or
 * reads the one of a new device, and asks the person about the new device; on a device with no identity it
 * shows its own name and fingerprint while it waits, asks for the code the other device shows, and asks whether to join the
 * account. Escape closes it, and closing it cancels the link.
 */
import { onMounted, ref, useId, useTemplateRef } from 'vue'
import { Button } from '@numen/ui'
import type { LinkRole } from '@/entities/identity'
import type { LinkCalls } from '../../lib/types'
import { useLinkSession } from '../../model/useLinkSession'
import { WORDS as words } from '../../words'
import LinkChoices from './LinkChoices.vue'
import LinkProgress from './LinkProgress.vue'
import LinkScanForm from './LinkScanForm.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ role: LinkRole; calls: LinkCalls }>()

const emit = defineEmits<{ (e: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const titleId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')
const isScanning = ref(false)
const { link, problem, isBusy, isChoosing, start, answers } = useLinkSession(() => props.calls, props.role)

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => dialog.value?.focus())

/* -------------------------------- Handlers -------------------------------- */
function onKey(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close')
}
</script>

<template>
  <div
    ref="dialog"
    class="link-dialog"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    tabindex="-1"
    @keydown="onKey"
  >
    <h2 :id="titleId" class="link-dialog__title">
      {{ role === 'host' ? words.deviceLinkNew : words.deviceJoin }}
    </h2>
    <template v-if="isChoosing">
      <LinkScanForm v-if="isScanning" :is-busy="isBusy" @submit="start('scan', $event)" />
      <LinkChoices v-else @show="start('show')" @scan="isScanning = true" />
    </template>
    <LinkProgress
      v-else-if="link"
      :link="link"
      :role="role"
      :is-busy="isBusy"
      @decide="answers.decide"
      @code="answers.sendCode"
      @join="answers.confirmJoin"
    />
    <p v-if="problem !== ''" role="alert">{{ problem }}</p>
    <div class="link-dialog__actions">
      <Button type="button" variant="outline" @click="emit('close')">{{ words.linkClose }}</Button>
    </div>
  </div>
</template>

<style scoped>
.link-dialog {
  position: fixed;
  inset: 10% 20%;
  z-index: var(--numen-lift-menu);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.25rem;
  overflow: auto;
  border: 1px solid var(--numen-field-border);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
}

.link-dialog__title {
  margin: 0;
  font-size: var(--numen-title-size);
  font-weight: 500;
}

.link-dialog__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
