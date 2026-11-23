<script setup lang="ts">
/**
 * The page of identity and licenses: the account identity of this device, then every license
 * declared by the installed plugins with the controls to activate or deactivate this machine.
 */
import { SettingsPage } from '@numen/plugin-manager'
import type { InstallationIdentity, InstallationLicenses } from '../../types'
import { useLicensesPage } from '../../model/useLicensesPage'
import { WORDS as words } from '../../words'
import LicenseCard from '../licenses/LicenseCard.vue'
import IdentitySection from './IdentitySection.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  identity: InstallationIdentity
  licenses: InstallationLicenses
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const page = useLicensesPage(() => props.licenses)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <SettingsPage :path="[words.settings, words.identityAndLicenses]" @back="onBack" @open="onBack">
    <div class="identity-page">
      <IdentitySection :identity="props.identity" />

      <div v-if="page.licenses.value.length === 0" class="identity-page__empty">
        <p class="identity-page__empty-title">{{ words.licensesEmpty }}</p>
        <p class="identity-page__empty-detail">{{ words.licensesEmptyDetail }}</p>
      </div>

      <LicenseCard
        v-for="license in page.licenses.value"
        :key="license.id"
        :license="license"
        :activate="page.activate"
        :deactivate="page.deactivate"
      />
    </div>
  </SettingsPage>
</template>

<style scoped>
.identity-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  inline-size: 100%;
  max-inline-size: var(--settings-measure);
  margin-inline: auto;
}

.identity-page__empty {
  padding: 1.5rem 1rem;
  border: var(--numen-stroke) dashed var(--numen-rule);
  border-radius: var(--numen-radius);
  text-align: center;
}

.identity-page__empty-title {
  margin: 0 0 0.25rem;
  font-weight: 500;
}

.identity-page__empty-detail {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
