<script setup lang="ts">
/**
 * The devices of the account: a list with the fingerprint of each (this device alone until the list arrives), the
 * way to link a new device by a QR code or to join an account with one, and the revoke of a device after a question.
 * Each device that is not this one has a Revoke button; this one has none.
 */
import { computed, ref } from 'vue'
import { Button, ConfirmDialog } from '@numen/ui'
import type { DeviceInfo, LinkRole } from '@/entities/identity'
import type { DeviceCalls } from '../../model/types'
import { useDevices } from '../../model/useDevices'
import { WORDS as words } from '../../words'
import { LinkDialog } from '../link-dialog'
import DeviceRow from './DeviceRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ calls: DeviceCalls; hasIdentity: boolean; current?: DeviceInfo }>()

/* --------------------------------- State ---------------------------------- */
const dialogRole = ref<LinkRole | undefined>(undefined)
const { asked, reason, since, question, problem, refresh, ask, cancel, confirmRevoke } = useDevices(
  () => props.calls,
  () => props.hasIdentity,
)
const shown = computed(() => {
  const listed = props.calls.list.value
  if (listed.length > 0 || props.current === undefined) return listed
  return [props.current]
})

const RECENT_MS = 7 * 24 * 60 * 60 * 1000
const now = ref(Date.now())

/** What a device did to another lately, so that the owner of a stolen device is told at once. */
const notices = computed(() =>
  shown.value.flatMap((device) => {
    const revocation = device.revocation
    const at = revocation === undefined ? Number.NaN : new Date(revocation.revokedAt).getTime()
    if (revocation === undefined || revocation.isByRecovery || Number.isNaN(at)) return []
    if (now.value - at > RECENT_MS || revocation.revokedByName === '') return []
    const date = new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(at)
    const text = words.deviceRevokeNotice(revocation.revokedByName, device.name, date)
    return [{ key: device.fingerprint, text }]
  }),
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
        <DeviceRow
          v-for="device in shown"
          :key="device.fingerprint"
          :device="device"
          @revoke="ask"
        />
      </ul>
      <p v-for="notice in notices" :key="notice.key" role="status" class="devices__notice">
        {{ notice.text }}
      </p>
      <p v-if="problem !== ''" role="alert" class="devices__problem">{{ problem }}</p>
    </template>

    <div class="devices__actions">
      <Button v-if="hasIdentity" type="button" variant="outline" @click="dialogRole = 'host'">
        {{ words.deviceLinkNew }}
      </Button>
      <Button v-else type="button" variant="outline" @click="dialogRole = 'joiner'">
        {{ words.deviceJoin }}
      </Button>
    </div>

    <LinkDialog
      v-if="dialogRole !== undefined"
      :role="dialogRole"
      :calls="calls.link"
      @close="onClose"
    />
    <ConfirmDialog
      v-if="asked !== undefined"
      tone="danger"
      :label="words.deviceRevokeTitle"
      :confirm-text="words.deviceRevoke"
      :cancel-text="words.deviceRevokeCancel"
      @confirm="confirmRevoke"
      @cancel="cancel"
    >
      <p class="devices__warning">{{ words.deviceRevokeWarning(asked.name) }}</p>
      <fieldset class="devices__why">
        <legend>{{ words.deviceRevokeWhy }}</legend>
        <label class="devices__choice">
          <input v-model="reason" type="radio" name="revoke-reason" value="lost" />
          {{ words.deviceRevokeLost }}
        </label>
        <label class="devices__choice">
          <input v-model="reason" type="radio" name="revoke-reason" value="stolen" />
          {{ words.deviceRevokeStolen }}
        </label>
        <label v-if="reason === 'stolen'" class="devices__choice">
          {{ words.deviceRevokeSince }}
          <input v-model="since" type="date" :aria-label="words.deviceRevokeSince" />
        </label>
        <p v-if="question !== ''" role="alert" class="devices__problem">{{ question }}</p>
      </fieldset>
    </ConfirmDialog>
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
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}

.devices__notice {
  margin: 0;
  padding: 0.5rem 0.75rem;
  border-radius: var(--numen-radius-tight);
  background-color: var(--settings-hover-bg);
  font-size: var(--numen-text-1);
  color: var(--numen-ink);
}

.devices__warning {
  margin: 0;
}

.devices__why {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin: 0;
  padding: 0;
  border: 0;
}

.devices__choice {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
