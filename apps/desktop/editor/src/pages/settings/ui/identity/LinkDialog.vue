<script setup lang="ts">
/**
 * The dialog that links a device to the account by a QR code. On a device that has the account it shows the QR code or
 * reads the one of a new device, and asks the person about the new device; on a device with no identity it
 * shows its own name and fingerprint while it waits, asks for the code the other device shows, and asks whether to join the
 * account. Escape closes it, and closing it cancels the link.
 */
import { onMounted, useId, useTemplateRef } from 'vue'
import { Button, QrCode, QrScanner } from '@numen/ui'
import type { LinkRole } from '@/entities/identity'
import { useLink } from '../../model/useLink'
import type { InstallationLink } from '../../types'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ role: LinkRole; link: InstallationLink; blocked?: string }>()

const emit = defineEmits<{ (e: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const titleId = useId()
const dialog = useTemplateRef<HTMLElement>('dialog')
const state = useLink(
  () => props.link,
  props.role,
  () => props.blocked ?? '',
)
const { link, problem, pastedText, typedCode, isBusy, isScanning } = state
const hasCamera = typeof navigator.mediaDevices?.getUserMedia === 'function'

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

    <div v-if="state.isChoosing.value && !isScanning" class="link-dialog__choices">
      <Button type="button" variant="outline" @click="state.start('show')">{{ words.linkShow }}</Button>
      <Button type="button" variant="outline" @click="isScanning = true">{{ words.linkScan }}</Button>
    </div>

    <form v-else-if="state.isChoosing.value" class="link-dialog__field" @submit.prevent="state.start('scan', pastedText)">
      <QrScanner v-if="hasCamera" @scanned="state.start('scan', $event)" />
      <label>
        <span>{{ words.linkPaste }}</span>
        <input v-model="pastedText" type="text" autocomplete="off" />
      </label>
      <Button type="submit" variant="solid" :disabled="isBusy">{{ words.linkUse }}</Button>
    </form>

    <template v-else-if="link">
      <dl v-if="role === 'joiner'" class="link-dialog__facts">
        <dt>{{ words.linkDeviceName }}</dt>
        <dd>{{ link.deviceName }}</dd>
        <dt>{{ words.linkDeviceFingerprint }}</dt>
        <dd class="link-dialog__mono">{{ link.deviceFingerprint }}</dd>
      </dl>

      <p v-if="link.phase === 'waiting_for_channel'">{{ words.linkPreparing }}</p>

      <template v-else-if="link.phase === 'waiting_for_peer' && link.mode === 'show'">
        <QrCode :text="link.qrText" :label="words.linkQr" />
        <label class="link-dialog__field">
          <span>{{ words.linkText }}</span>
          <input type="text" readonly :value="link.qrText" />
        </label>
        <dl v-if="role === 'host'" class="link-dialog__facts">
          <dt>{{ words.identityAccount }}</dt>
          <dd class="link-dialog__mono">{{ link.accountFingerprint }}</dd>
        </dl>
      </template>

      <p v-else-if="link.phase === 'waiting_for_peer' || link.phase === 'waiting_for_accept'">
        {{ words.linkWaiting }}
      </p>

      <template v-else-if="link.phase === 'confirm_peer'">
        <dl class="link-dialog__facts">
          <dt>{{ words.linkDeviceName }}</dt>
          <dd>{{ link.peerName }}</dd>
          <dt>{{ words.linkDeviceFingerprint }}</dt>
          <dd class="link-dialog__mono">{{ link.peerFingerprint }}</dd>
          <template v-if="link.code !== ''">
            <dt>{{ words.linkCode }}</dt>
            <dd class="link-dialog__mono">{{ link.code }}</dd>
          </template>
        </dl>
        <p>{{ words.linkCompare }}</p>
        <p v-if="link.isApproved">{{ words.linkWaitingCode }}</p>
        <div v-else class="link-dialog__actions">
          <Button type="button" variant="solid" :disabled="isBusy" @click="state.decide(true)">
            {{ words.linkApprove }}
          </Button>
          <Button type="button" variant="outline" :disabled="isBusy" @click="state.decide(false)">
            {{ words.linkDecline }}
          </Button>
        </div>
      </template>

      <form v-else-if="link.phase === 'enter_code'" class="link-dialog__field" @submit.prevent="state.sendCode()">
        <label>
          <span>{{ words.linkCodeField }}</span>
          <input v-model="typedCode" type="text" inputmode="numeric" maxlength="6" autocomplete="off" />
        </label>
        <p v-if="link.triesLeft > 0 && link.triesLeft < 3">{{ link.triesLeft }}</p>
        <Button type="submit" variant="solid" :disabled="isBusy">{{ words.linkCodeConfirm }}</Button>
      </form>

      <template v-else-if="link.phase === 'confirm_account' && role === 'joiner'">
        <p>{{ words.linkJoinQuestion(link.accountFingerprint) }}</p>
        <div class="link-dialog__actions">
          <Button type="button" variant="solid" :disabled="isBusy" @click="state.confirmJoin(true)">
            {{ words.linkJoin }}
          </Button>
          <Button type="button" variant="outline" :disabled="isBusy" @click="state.confirmJoin(false)">
            {{ words.linkCancel }}
          </Button>
        </div>
      </template>

      <p v-else-if="link.phase === 'linked'">{{ words.linkDone }}</p>
      <p v-else-if="link.phase === 'failed'" role="alert">{{ words.linkReason(link.reason) }}</p>
      <p v-else>{{ words.linkLinking }}</p>
    </template>

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
  z-index: 50;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1.25rem;
  overflow: auto;
  background: var(--numen-surface);
  border: 1px solid var(--numen-hairline, currentcolor);
  border-radius: 0.75rem;
}

.link-dialog__title {
  margin: 0;
  font-size: var(--numen-text-2, 1.1rem);
}

.link-dialog__choices,
.link-dialog__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.link-dialog__facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.375rem 1rem;
  margin: 0;
}

.link-dialog__facts dd {
  margin: 0;
}

.link-dialog__mono {
  font-family: var(--numen-font-mono);
}

.link-dialog__field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
</style>
