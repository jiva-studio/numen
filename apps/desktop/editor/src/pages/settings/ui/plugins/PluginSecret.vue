<script setup lang="ts">
/**
 * The field of one secret a plugin declares, wired to the secrets of the installation: a click on
 * a secret in its menu allows or denies the declaration, and the button opens the page of secrets.
 */
import { computed } from 'vue'
import type { PluginInfo, SecretDeclaration, Secret } from '@/entities/settings'
import { getDeclaration } from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { SecretField } from '../secret-field'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
  secrets: readonly SecretDeclaration[]
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const sections = computed(() => installation.value.secrets.sections.value)
const menus = computed(() => installation.value.secrets.menus.value)
/* -------------------------------- Handlers -------------------------------- */
function onToggle(secret: SecretDeclaration, chosen: Secret, isAllowed: boolean): void {
  const { pluginId, name, group } = getDeclaration(props.plugin, secret)
  if (isAllowed)
    installation.value.secrets.access.allowPluginSecret(chosen.id, pluginId, name, group)
  else installation.value.secrets.access.denyPluginSecret(chosen.id, pluginId, name, group)
}

function onOpenSecrets(): void {
  props.state.initialPluginId.value = 'secrets'
}
</script>

<template>
  <SecretField
    v-for="secret in secrets"
    :key="`${plugin.id}-${secret.name}-${secret.group}`"
    :at="`key-${plugin.id}-${secret.name}`"
    :name="secret.title"
    :detail="secret.description"
    :declaration="getDeclaration(plugin, secret)"
    :sections="sections"
    :menus="menus"
    @toggle="(chosen, isAllowed) => onToggle(secret, chosen, isAllowed)"
    @open-secrets="onOpenSecrets"
  />
</template>
