<script setup lang="ts">
/**
 * The page of identity and licenses: the account identity of this device, the views plugins registered for
 * this page, then every license declared by the installed plugins with the controls to activate or
 * deactivate this machine.
 */
import { SettingsPage } from '@numen/plugin-manager'
import type { InstallationIdentity, InstallationLicenses, InstallationPlugin } from '../../types'
import { useIdentityPage } from '../../model/useIdentityPage'
import { WORDS as words } from '../../words'
import LicenseCard from '../licenses/LicenseCard.vue'
import IdentitySection from './IdentitySection.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  identity: InstallationIdentity
  licenses: InstallationLicenses
  plugin: Pick<InstallationPlugin, 'getIdentityViews'>
}>()

const emit = defineEmits<{
  (e: 'back'): void
}>()

/* --------------------------------- State ---------------------------------- */
const page = useIdentityPage(
  () => props.licenses,
  () => props.plugin,
)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}
</script>

<template>
  <SettingsPage :path="[words.settings, words.identityAndLicenses]" @back="onBack" @open="onBack">
    <div class="identity-page">
      <IdentitySection :identity="props.identity" />

      <component
        :is="view.component"
        v-for="view in page.views.value"
        :key="view.pluginId + '/' + view.id"
        :plugin-id="view.pluginId"
      />

      <div
        v-if="page.licenses.value.length === 0 && page.views.value.length === 0"
        class="identity-page__empty"
      >
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
