<script setup lang="ts">
/** One device of the account: its name and fingerprint, whether it is this one or revoked, and a Revoke button for another. */
import { Button } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{ device: DeviceInfo }>()

const emit = defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()
</script>

<template>
  <li class="device-row">
    <span>{{ device.name }}</span>
    <span class="device-row__fingerprint">{{ device.fingerprint }}</span>
    <span v-if="device.isThis" class="device-row__mark">{{ words.deviceThis }}</span>
    <span v-if="device.isRevoked" class="device-row__mark">{{ words.deviceRevoked }}</span>
    <Button
      v-if="!device.isThis && !device.isRevoked"
      type="button"
      variant="outline"
      @click="emit('revoke', device)"
    >
      {{ words.deviceRevoke }}
    </Button>
  </li>
</template>

<style scoped>
.device-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem 0.75rem;
  align-items: center;
}

.device-row__fingerprint {
  font-family: var(--numen-font-mono);
}

.device-row__mark {
  color: var(--numen-hushed);
}
</style>
