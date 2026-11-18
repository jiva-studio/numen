<script setup lang="ts">
/**
 * The page of licenses: every license declared by the installed plugins, with the plugins that
 * share it, the providers where a key can be purchased, and the controls to activate or deactivate
 * this machine.
 */
import { onBeforeUnmount, onMounted } from 'vue'
import { SettingsPage } from '@numen/plugin-manager'
import { runInBackground } from '@numen/ui'
import type { InstallationLicenses } from '../../types'
import { WORDS as words } from '../../words'
import LicenseCard from './LicenseCard.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ licenses: InstallationLicenses }>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const watchAbort = new AbortController()

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  runInBackground(props.licenses.load(), () => undefined)
  runInBackground(props.licenses.watch(watchAbort.signal), () => undefined)
})

onBeforeUnmount(() => {
  watchAbort.abort()
})

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <SettingsPage :path="[words.settings, words.licenses]" @back="onBack" @open="onBack">
    <div class="licenses-page">
      <div v-if="licenses.list.value.length === 0" class="licenses-page__empty">
        <p class="licenses-page__empty-title">{{ words.licensesEmpty }}</p>
        <p class="licenses-page__empty-detail">{{ words.licensesEmptyDetail }}</p>
      </div>

      <LicenseCard
        v-for="license in licenses.list.value"
        :key="license.id"
        :license="license"
        :activate="licenses.activate"
        :deactivate="licenses.deactivate"
      />
    </div>
  </SettingsPage>
</template>

<style scoped>
.licenses-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.licenses-page__empty {
  padding: 1.5rem 1rem;
  border: var(--numen-stroke) dashed var(--numen-rule);
  border-radius: var(--numen-radius);
  text-align: center;
}

.licenses-page__empty-title {
  margin: 0 0 0.25rem;
  font-weight: 500;
}

.licenses-page__empty-detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
