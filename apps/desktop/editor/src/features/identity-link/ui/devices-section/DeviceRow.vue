<script setup lang="ts">
/**
 * One device of the identity: its name, when it was linked, its fingerprint and whether it is this
 * one. Every row has the same columns; the actions column holds a "..." menu with Remove on another
 * device and stays empty on this one.
 */
import { computed } from 'vue'
import { NavigationRow } from '@numen/plugin-manager'
import { Fingerprint } from '@numen/ui'
import type { DeviceInfo } from '@/entities/identity'
import { WORDS as words } from '../../words'
import DeviceActions from './DeviceActions.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ device: DeviceInfo }>()

defineEmits<{ (e: 'revoke', device: DeviceInfo): void }>()

/* --------------------------------- State ---------------------------------- */
const date = computed(() => {
  const added = new Date(props.device.addedAt)
  if (Number.isNaN(added.getTime())) return ''
  return new Intl.DateTimeFormat('en', { dateStyle: 'long', timeZone: 'UTC' }).format(added)
})

const hasActions = computed(() => !props.device.isThis && !props.device.isRevoked)
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
      </template>
      <template #trail>
        <Fingerprint class="device-row__fingerprint" :value="device.fingerprint" />
        <span class="device-row__actions">
          <DeviceActions v-if="hasActions" :device="device" @revoke="$emit('revoke', $event)" />
        </span>
      </template>
    </NavigationRow>
  </li>
</template>

<style scoped>
.device-row__fingerprint {
  flex: 0 1 auto;
  min-inline-size: 8rem;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  letter-spacing: 0.02em;
}

.device-row__actions {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
}

.device-row__badge {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
}
</style>
