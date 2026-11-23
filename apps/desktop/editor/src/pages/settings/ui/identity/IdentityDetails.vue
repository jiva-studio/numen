<script setup lang="ts">
/**
 * The identity of this device as two cards: its fingerprints with the way to delete it, after a
 * confirmation, and the list of devices it is linked to.
 */
import { computed } from 'vue'
import { Button, IslandCard } from '@numen/ui'
import { NavigationRow } from '@numen/plugin-manager'
import type { IdentityInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '../confirm-dialog'
import SettingRow from '../setting-row/SettingRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  info: IdentityInfo
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
</script>

<template>
  <IslandCard :title="words.identitySection">
    <template v-if="date !== ''" #aside>{{ words.identityCreatedOn(date) }}</template>
    <SettingRow
      at="identity-account"
      :name="words.identityAccount"
      :detail="words.identityAccountDetail"
    >
      <span class="identity-details__mono">{{ info.accountFingerprint }}</span>
    </SettingRow>
    <SettingRow
      at="identity-device"
      :name="words.identityThisDevice"
      :detail="date === '' ? info.deviceName : `${info.deviceName} · ${date}`"
    >
      <span class="identity-details__mono">{{ info.deviceFingerprint }}</span>
    </SettingRow>
    <SettingRow
      at="identity-remove"
      :name="words.identityRemoveName"
      :detail="words.identityRemoveDetail"
    >
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('ask')">
        {{ words.identityDelete }}
      </Button>
    </SettingRow>
    <p v-if="problem !== ''" role="alert" class="identity-details__problem">{{ problem }}</p>
  </IslandCard>

  <IslandCard class="identity-details__devices" :title="words.identityDevices">
    <template #aside>{{ words.identityDeviceCount(1) }}</template>
    <NavigationRow
      class="identity-details__current"
      category="device"
      is-static
      :name="info.deviceName"
      :detail="words.identityDeviceDetail(date, info.deviceFingerprint)"
    >
      <template #trail>
        <span class="identity-details__state">{{ words.identityDeviceNow }}</span>
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
}

.identity-details__devices :deep(.island-card__body) {
  padding: 0;
}

.identity-details__current,
.identity-details__current:hover {
  background-color: var(--settings-hover-bg);
}

.identity-details__state {
  color: var(--numen-hushed);
}

.identity-details__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
