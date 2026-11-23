<script setup lang="ts">
/**
 * The identity of this device as two cards: the account fingerprint, the rows the default slot adds
 * and the way to delete the identity after a confirmation, and the list of the devices of the account
 * with the link of a new one and the revoke of another. A revoked device shows only itself.
 */
import { computed } from 'vue'
import { Trash2 } from '@lucide/vue'
import { ConfirmDialog, IslandCard } from '@numen/ui'
import { NavigationRow } from '@numen/plugin-manager'
import type { DeviceInfo, IdentityInfo } from '@/entities/identity'
import { DevicesSection, type DeviceCalls } from '@/features/identity-link'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'
import CardAction from './CardAction.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  info: IdentityInfo
  devices: DeviceCalls
  problem: string
  isBusy: boolean
  isAskingDelete: boolean
}>()

const emit = defineEmits<{
  (e: 'ask'): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const date = computed(() => {
  const added = new Date(props.info.deviceAddedAt)
  if (Number.isNaN(added.getTime())) return ''
  return new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(added)
})
const thisDevice = computed<DeviceInfo>(() => ({
  name: props.info.deviceName,
  fingerprint: props.info.deviceFingerprint,
  addedAt: props.info.deviceAddedAt,
  isThis: true,
  isRevoked: false,
}))
const count = computed(() =>
  props.info.isRevoked ? 1 : Math.max(props.devices.list.value.length, 1),
)
</script>

<template>
  <IslandCard :title="words.identitySection">
    <template #aside>
      <CardAction :is-disabled="isBusy" @click="emit('ask')">
        <Trash2 :size="14" aria-hidden="true" />
        {{ info.isRevoked ? words.deviceDeleteRevoked : words.identityDelete }}
      </CardAction>
    </template>
    <SettingRow
      class="identity-details__fact"
      at="identity-account"
      :name="words.identityAccount"
      :detail="words.identityAccountDetail"
    >
      <span class="identity-details__mono">{{ info.accountFingerprint }}</span>
    </SettingRow>
    <slot />
    <p v-if="info.isRevoked" role="alert" class="identity-details__revoked">
      {{ words.deviceRevokedNotice }}
    </p>
    <p v-if="problem !== ''" role="alert" class="identity-details__problem">{{ problem }}</p>
  </IslandCard>

  <IslandCard class="identity-details__devices" :title="words.identityDevices">
    <template #aside>{{ words.identityDeviceCount(count) }}</template>
    <DevicesSection v-if="!info.isRevoked" :calls="devices" has-identity :current="thisDevice" />
    <NavigationRow
      v-else
      class="identity-details__current"
      category="device"
      is-static
      :name="info.deviceName"
      :detail="words.identityDeviceLinked(date)"
    >
      <template #name-suffix>
        <span class="identity-details__badge">{{ words.identityThisDevice }}</span>
      </template>
      <template #trail>
        <span class="identity-details__mono">{{ info.deviceFingerprint }}</span>
      </template>
    </NavigationRow>
  </IslandCard>

  <ConfirmDialog
    v-if="isAskingDelete"
    tone="danger"
    :text="words.identityDeleteWarning"
    :label="words.identityDeleteTitle"
    :confirm-text="words.identityDeleteConfirm"
    :cancel-text="words.identityDeleteCancel"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  />
</template>

<style scoped>
.identity-details__mono {
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.identity-details__fact.identity-details__fact {
  grid-template-columns: minmax(0, 1fr) max-content;
}

.identity-details__devices :deep(.island-card__body) {
  padding: 0;
}

.identity-details__devices :deep(.devices__actions),
.identity-details__devices :deep(.devices__problem) {
  padding: 0 1rem 0.75rem;
}

.identity-details__current,
.identity-details__current:hover {
  background-color: var(--settings-hover-bg);
}

.identity-details__badge {
  padding: 0.125rem 0.5rem;
  border-radius: var(--numen-radius-tight);
  background-color: var(--settings-hover-bg);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
}

.identity-details__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
