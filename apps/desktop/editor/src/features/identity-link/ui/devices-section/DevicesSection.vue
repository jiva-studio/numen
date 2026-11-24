<script setup lang="ts">
/**
 * The devices of the account: a list with the fingerprint of each (this device alone until the list arrives), the
 * dialog that links a new device by a QR code when `isLinking` is set, and the revoke of a device after a question.
 * Each device that is not this one has a Remove button; this one has none.
 */
import { computed } from 'vue'
import { ConfirmDialog } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import type { DeviceCalls } from '../../model/types'
import { useDevices } from '../../model/useDevices'
import { WORDS as words } from '../../words'
import { LinkDialog } from '../link-dialog'
import DeviceRow from './DeviceRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ calls: DeviceCalls; current?: DeviceInfo; isLinking: boolean }>()

const emit = defineEmits<{ (e: 'update:isLinking', value: boolean): void }>()

/* --------------------------------- State ---------------------------------- */
const { asked, problem, refresh, confirmRevoke } = useDevices(() => props.calls)
const shown = computed(() => {
  const listed = props.calls.list.value
  if (listed.length > 0 || props.current === undefined) return listed
  return [props.current]
})

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  emit('update:isLinking', false)
  refresh()
}
</script>

<template>
  <section class="devices" :aria-label="words.devices">
    <ul class="devices__list" :aria-label="words.devices">
      <DeviceRow
        v-for="device in shown"
        :key="device.fingerprint"
        :device="device"
        @revoke="asked = $event"
      />
    </ul>
    <p v-if="problem !== ''" role="alert" class="devices__problem">{{ problem }}</p>

    <LinkDialog
      v-if="isLinking"
      :calls="calls.link"
      @close="onClose"
    />
    <ConfirmDialog
      v-if="asked !== undefined"
      tone="danger"
      :text="words.deviceRevokeWarning(asked.name)"
      :label="words.deviceRevokeTitle"
      :confirm-text="words.deviceRevoke"
      :cancel-text="words.deviceRevokeCancel"
      @confirm="confirmRevoke"
      @cancel="asked = undefined"
    />
  </section>
</template>

<style scoped>
.devices {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.devices__list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.devices__problem {
  margin: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
