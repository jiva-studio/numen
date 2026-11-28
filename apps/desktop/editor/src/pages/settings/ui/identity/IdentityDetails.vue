<script setup lang="ts">
/**
 * The identity of this device as two cards: the identity fingerprint, the rows the default slot adds
 * and the way to delete the identity after a confirmation, and the list of the devices of the identity
 * with the link of a new one and the revoke of another. A revoked device shows only itself.
 */
import { computed, ref } from 'vue'
import { Link, Trash2 } from '@lucide/vue'
import {
  CardAction,
  CardActions,
  ConfirmDialog,
  Fingerprint,
  IslandCard,
  type CardActionItem,
} from '@numen/ui'
import { NavigationRow } from '@numen/plugin-manager'
import type { DeviceInfo, IdentityInfo } from '@/entities/identity'
import { DevicesSection, type DeviceCalls } from '@/features/identity-link'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

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
const isLinking = ref(false)
const accountActions = computed<readonly CardActionItem[]>(() => [
  {
    id: 'delete',
    label: props.info.isRevoked ? words.deviceDeleteRevoked : words.identityDelete,
    icon: Trash2,
    isDisabled: props.isBusy,
  },
])
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
</script>

<template>
  <IslandCard :title="words.identitySection">
    <template #aside>
      <CardActions
        :items="accountActions"
        :more-label="words.moreActions"
        @choose="() => emit('ask')"
      />
    </template>
    <SettingRow
      class="identity-details__fact"
      at="identity-account"
      :name="words.identityAccount"
      :detail="words.identityAccountDetail"
    >
      <Fingerprint class="identity-details__mono" :value="info.accountFingerprint" />
    </SettingRow>
    <slot />
    <p v-if="info.isRevoked" role="alert" class="identity-details__revoked">
      {{ words.deviceRevokedNotice }}
    </p>
    <p v-if="problem !== ''" role="alert" class="identity-details__problem">{{ problem }}</p>
  </IslandCard>

  <IslandCard class="identity-details__devices" :title="words.identityDevices">
    <template #aside>
      <CardAction
        v-if="!info.isRevoked"
        is-primary
        :label="words.identityLinkDevice"
        @click="isLinking = true"
      >
        <Link :size="14" aria-hidden="true" />
      </CardAction>
    </template>
    <DevicesSection
      v-if="!info.isRevoked"
      v-model:is-linking="isLinking"
      :calls="devices"
      :current="thisDevice"
    />
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
        <Fingerprint class="identity-details__mono" :value="info.deviceFingerprint" />
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
  flex: 0 1 auto;
  min-inline-size: 8rem;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
}

.identity-details__fact.identity-details__fact {
  grid-template-columns: minmax(min-content, 1fr) minmax(8rem, max-content);
}

.identity-details__fact :deep(.settings__name) {
  overflow-wrap: normal;
}

.identity-details__fact :deep(.settings__detail) {
  white-space: normal;
}

.identity-details__devices :deep(.island-card__body) {
  padding: 0;
}

.identity-details__devices :deep(.devices__problem) {
  padding: 0 1rem 0.75rem;
}

.identity-details__badge {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
}

.identity-details__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
