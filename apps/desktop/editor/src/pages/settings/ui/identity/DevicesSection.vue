<script setup lang="ts">
/**
 * The devices of the account: a list with the fingerprint of each, the way to link a new device by a QR code or to join an
 * account with one, and the revoke of a device after a question. Each device that is not this one has a Revoke
 * button; this one has none.
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Button, runInBackground } from '@numen/ui'
import type { DeviceInfo, LinkRole } from '@/entities/identity'
import type { InstallationIdentity } from '../../types'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import LinkDialog from './LinkDialog.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ identity: InstallationIdentity; hasIdentity: boolean }>()

/* --------------------------------- State ---------------------------------- */
const REFRESH_MS = 2000
const dialogRole = ref<LinkRole | undefined>(undefined)
const revoking = ref<DeviceInfo | undefined>(undefined)
const problem = ref('')
let timer: ReturnType<typeof setInterval> | undefined

/* ---------------------------------- Hooks --------------------------------- */
onMounted(() => {
  refresh()
  timer = setInterval(refresh, REFRESH_MS)
})

onBeforeUnmount(() => {
  if (timer !== undefined) clearInterval(timer)
})

/* -------------------------------- Handlers -------------------------------- */
function refresh(): void {
  if (!props.hasIdentity) return
  runInBackground(props.identity.loadDevices(), () => undefined)
}

function onClose(): void {
  dialogRole.value = undefined
  refresh()
}

async function onRevoke(): Promise<void> {
  const device = revoking.value
  revoking.value = undefined
  if (device === undefined) return
  const answer = await props.identity.revoke(device.fingerprint)
  problem.value = answer.ok ? '' : answer.error
}
</script>

<template>
  <section class="devices" :aria-label="words.devices">
    <template v-if="hasIdentity">
      <ul class="devices__list" :aria-label="words.devices">
        <li v-for="device in identity.devices.value" :key="device.fingerprint" class="devices__row">
          <span>{{ device.name }}</span>
          <span class="devices__fingerprint">{{ device.fingerprint }}</span>
          <span v-if="device.isThis" class="devices__mark">{{ words.deviceThis }}</span>
          <span v-if="device.isRevoked" class="devices__mark">{{ words.deviceRevoked }}</span>
          <Button
            v-if="!device.isThis && !device.isRevoked"
            type="button"
            variant="outline"
            @click="revoking = device"
          >
            {{ words.deviceRevoke }}
          </Button>
        </li>
      </ul>
      <p v-if="problem !== ''" role="alert">{{ problem }}</p>
    </template>

    <div class="devices__actions">
      <Button v-if="hasIdentity" type="button" variant="outline" @click="dialogRole = 'host'">
        {{ words.deviceLinkNew }}
      </Button>
      <Button type="button" variant="outline" @click="dialogRole = 'joiner'">
        {{ words.deviceJoin }}
      </Button>
    </div>

    <LinkDialog
      v-if="dialogRole !== undefined"
      :role="dialogRole"
      :link="identity.link"
      :blocked="dialogRole === 'joiner' && hasIdentity ? words.linkReason('device_has_identity') : ''"
      @close="onClose"
    />
    <ConfirmDialog
      v-if="revoking !== undefined"
      tone="danger"
      :text="words.deviceRevokeWarning(revoking.name)"
      :label="words.deviceRevokeTitle"
      :confirm-text="words.deviceRevoke"
      :cancel-text="words.deviceRevokeCancel"
      @confirm="onRevoke"
      @cancel="revoking = undefined"
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
  gap: 0.375rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.devices__row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.75rem;
  align-items: center;
}

.devices__fingerprint {
  font-family: var(--numen-font-mono);
}

.devices__mark {
  color: var(--numen-hushed);
}

.devices__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
