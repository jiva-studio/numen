<script setup lang="ts">
/**
 * One device of the account: its name, when it was linked and its fingerprint, whether it is this
 * one or revoked, and a Revoke button for another.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import { NavigationRow } from '@numen/plugin-manager'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ device: DeviceInfo }>()

const emit = defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()

/* --------------------------------- State ---------------------------------- */
const date = computed(() => {
  const added = new Date(props.device.addedAt)
  if (Number.isNaN(added.getTime())) return ''
  return new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(added)
})
</script>

<template>
  <li class="device-row">
    <NavigationRow
      category="device"
      is-static
      :name="device.name"
      :detail="words.deviceLinked(date)"
    >
      <template #name-suffix>
        <span v-if="device.isThis" class="device-row__badge">{{ words.deviceThis }}</span>
        <span v-if="device.isRevoked" class="device-row__badge">{{ words.deviceRevoked }}</span>
      </template>
      <template #trail>
        <span class="device-row__fingerprint">{{ device.fingerprint }}</span>
        <Button
          v-if="!device.isThis && !device.isRevoked"
          type="button"
          variant="outline"
          @click="emit('revoke', device)"
        >
          {{ words.deviceRevoke }}
        </Button>
      </template>
    </NavigationRow>
  </li>
</template>

<style scoped>
.device-row__fingerprint {
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.device-row__badge {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
}
</style>
