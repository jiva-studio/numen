<script setup lang="ts">
/**
 * The devices of the account: a list with the fingerprint of each (this device alone until the list arrives), the
 * way to link a new device by a QR code, and the revoke of a device after a question.
 * Each device that is not this one has a Revoke button; this one has none.
 */
import { computed, ref } from 'vue'
import { Button, ConfirmDialog } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import type { DeviceCalls } from '../../model/types'
import { useDevices } from '../../model/useDevices'
import { WORDS as words } from '../../words'
import { LinkDialog } from '../link-dialog'
import DeviceRow from './DeviceRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ calls: DeviceCalls; current?: DeviceInfo }>()

/* --------------------------------- State ---------------------------------- */
const isLinking = ref(false)
const { asked, problem, refresh, confirmRevoke } = useDevices(() => props.calls)
const shown = computed(() => {
  const listed = props.calls.list.value
  if (listed.length > 0 || props.current === undefined) return listed
  return [props.current]
})

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  isLinking.value = false
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

    <div class="devices__actions">
      <Button type="button" variant="outline" @click="isLinking = true">
        {{ words.deviceLinkNew }}
      </Button>
    </div>

    <LinkDialog
      v-if="isLinking"
      role="host"
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

.devices__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.5rem;
}

.devices__problem {
  margin: 0;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
