<script setup lang="ts">
/**
 * The devices of the identity: a list with the fingerprint of each (this device alone until the list arrives), the
 * dialog that links a new device by a QR code when `isLinking` is set, and the revoke of a device after a question.
 * A right-click on a device that is not this one opens a menu with Remove; a hint says so.
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

const emit = defineEmits<{ (e: 'update:is-linking', value: boolean): void }>()

/* --------------------------------- State ---------------------------------- */
const { asked, problem, refresh, confirmRevoke } = useDevices(() => props.calls)
const shown = computed(() => {
  const listed = props.calls.list.value
  if (listed.length > 0 || props.current === undefined) return listed
  return [props.current]
})

const hasOther = computed(() => shown.value.some((device) => !device.isThis && !device.isRevoked))

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  emit('update:is-linking', false)
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
    <p v-if="hasOther" class="devices__hint">{{ words.devicesHint }}</p>
    <p v-if="problem !== ''" role="alert" class="devices__problem">{{ problem }}</p>

    <LinkDialog
      v-if="isLinking"
      :calls="calls.link"
      @close="onClose"
    />
    <ConfirmDialog
      v-if="asked !== undefined"
      tone="danger"
      :text="words.deviceRevokeWarning(asked.name, asked.fingerprint)"
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

.devices__hint {
  margin: 0;
  padding: 0 1rem 0.75rem;
  font-size: var(--numen-text-2);
  color: var(--numen-hushed);
}

.devices__problem {
  margin: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
