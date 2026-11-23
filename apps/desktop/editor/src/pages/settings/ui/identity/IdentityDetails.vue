<script setup lang="ts">
/**
 * The identity of this device: its fingerprints and the way to delete it, after a confirmation.
 */
import { Button } from '@numen/ui'
import type { IdentityInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ info: IdentityInfo; isBusy: boolean; isAskingDelete: boolean }>()

const emit = defineEmits<{
  (e: 'ask'): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <dl class="identity-details__facts">
    <dt>{{ words.identityAccount }}</dt>
    <dd class="identity-details__fingerprint">{{ info.accountFingerprint }}</dd>
    <dt>{{ words.identityThisDevice }}</dt>
    <dd>
      <span>{{ info.deviceName }}</span>
      <span class="identity-details__fingerprint">{{ info.deviceFingerprint }}</span>
    </dd>
  </dl>
  <p v-if="info.isRevoked" role="alert" class="identity-details__revoked">
    {{ words.deviceRevokedNotice }}
  </p>
  <div class="identity-details__actions">
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('ask')">
      {{ info.isRevoked ? words.deviceDeleteRevoked : words.identityDelete }}
    </Button>
  </div>
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
.identity-details__facts {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.375rem 1rem;
  margin: 0;
}

.identity-details__facts dt {
  color: var(--numen-hushed);
}

.identity-details__facts dd {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.75rem;
  margin: 0;
}

.identity-details__fingerprint {
  font-family: var(--numen-font-mono);
}

.identity-details__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
