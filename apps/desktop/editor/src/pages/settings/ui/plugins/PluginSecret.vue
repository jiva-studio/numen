<script setup lang="ts">
/**
 * The field of one secret a plugin declares, wired to the secrets of the installation: a click on
 * a secret in its menu allows or denies the declaration, and the button opens the page of secrets.
 */
import { computed } from 'vue'
import type { PluginInfo, PluginSecret, Secret } from '@/entities/settings'
import { createSecretSections, getDeclaration } from '../../lib/secretSections'
import type { SettingsTabState } from '../../types'
import { SecretField } from '../secret-field'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
  secrets: readonly PluginSecret[]
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const sections = computed(() =>
  createSecretSections(
    installation.value.plugins?.value ?? [],
    installation.value.secrets?.value ?? [],
  ),
)
/* -------------------------------- Handlers -------------------------------- */
function onToggle(secret: PluginSecret, chosen: Secret, isAllowed: boolean): void {
  const { pluginId, name, group } = getDeclaration(props.plugin, secret)
  if (isAllowed) installation.value.allowPluginSecret(chosen.id, pluginId, name, group)
  else installation.value.denyPluginSecret(chosen.id, pluginId, name, group)
}

function onOpenSecrets(): void {
  const target = props.state.initialPluginId
  if (target) target.value = 'secrets'
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
    @toggle="(chosen, isAllowed) => onToggle(secret, chosen, isAllowed)"
    @open-secrets="onOpenSecrets"
  />
</template>
