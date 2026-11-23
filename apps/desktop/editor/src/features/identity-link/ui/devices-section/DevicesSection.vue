<script setup lang="ts">
/**
 * The devices of the account: a list with the fingerprint of each, the way to link a new device by a QR code or to join an
 * account with one, and the revoke of a device after a question. Each device that is not this one has a Revoke
 * button; this one has none.
 */
import { ref } from 'vue'
import { Button } from '@numen/ui'
import type { LinkRole } from '@/entities/identity'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import type { DeviceCalls } from '../../lib/types'
import { useDevices } from '../../model/useDevices'
import { WORDS as words } from '../../words'
import { LinkDialog } from '../link-dialog'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ calls: DeviceCalls; hasIdentity: boolean }>()

/* --------------------------------- State ---------------------------------- */
const dialogRole = ref<LinkRole | undefined>(undefined)
const { asked, problem, refresh, confirmRevoke } = useDevices(
  () => props.calls,
  () => props.hasIdentity,
)

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  dialogRole.value = undefined
  refresh()
}
</script>

<template>
  <section class="devices" :aria-label="words.devices">
    <template v-if="hasIdentity">
      <ul class="devices__list" :aria-label="words.devices">
        <li v-for="device in calls.devices.value" :key="device.fingerprint" class="devices__row">
          <span>{{ device.name }}</span>
          <span class="devices__fingerprint">{{ device.fingerprint }}</span>
          <span v-if="device.isThis" class="devices__mark">{{ words.deviceThis }}</span>
          <span v-if="device.isRevoked" class="devices__mark">{{ words.deviceRevoked }}</span>
          <Button v-if="!device.isThis && !device.isRevoked" type="button" variant="outline" @click="asked = device">
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
      <Button v-else type="button" variant="outline" @click="dialogRole = 'joiner'">
        {{ words.deviceJoin }}
      </Button>
    </div>

    <LinkDialog v-if="dialogRole !== undefined" :role="dialogRole" :calls="calls.link" @close="onClose" />
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
