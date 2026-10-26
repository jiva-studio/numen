<script setup lang="ts">
/**
 * The agent that answers the plugins asking a model, whether each plugin may
 * ask at all, and the keys of the secrets plugins declare.
 */
import { computed } from 'vue'
import { Button, Switch } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import { AgentChoice } from './agent-choice'
import { SecretField } from './secret-field'
import type { SettingsTabState } from '../types'
import type { CompletionChoice, PluginInfo } from '@/entities/settings'
import { isShown, write } from '@/entities/settings'
import {
  DEFAULT_AGENT_AT,
  DEFAULT_MODEL_AT,
  createAgentModelChoices,
  createDefaultAgentChoices,
  findAgent,
  getAgents,
  getAskingPlugins,
  getPluginsWithSecrets,
} from '../lib/agents'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const plugins = computed<readonly PluginInfo[]>(() => installation.value.plugins?.value ?? [])

const defaults = computed<CompletionChoice>(() => {
  const agent = installation.value.getSetting(DEFAULT_AGENT_AT)
  const model = installation.value.getSetting(DEFAULT_MODEL_AT)
  return {
    agent: typeof agent === 'string' ? agent : '',
    model: typeof model === 'string' ? model : '',
  }
})
const agents = computed(() => getAgents(installation.value))
const agentChoices = computed(() =>
  createDefaultAgentChoices(agents.value, defaults.value.agent, words),
)
const modelChoices = computed(() =>
  createAgentModelChoices(
    findAgent(agents.value, defaults.value.agent),
    defaults.value.model,
    words,
  ),
)
const askingPlugins = computed(() => getAskingPlugins(plugins.value))
const waitingPlugins = computed(() => askingPlugins.value.filter((one) => one.consent === 'asked'))
const pluginsWithSecrets = computed(() => getPluginsWithSecrets(plugins.value))

/** The secrets of the plugin whose condition holds. */
function getShownSecrets(plugin: PluginInfo) {
  const read = (key: string) => installation.value.getSetting(['plugins', plugin.id, key])
  return plugin.secrets.filter((one) => isShown(plugin, one.when, read))
}

/* -------------------------------- Handlers -------------------------------- */
function onDefaultChoose(choice: CompletionChoice) {
  installation.value.write([
    { at: DEFAULT_AGENT_AT, value: write(choice.agent) },
    { at: DEFAULT_MODEL_AT, value: write(choice.model) },
  ])
}

function onConsentChange(pluginId: string, isAllowed: boolean) {
  installation.value.write([{ at: ['plugins', pluginId, 'aiConsent'], value: write(isAllowed) }])
}

function onSecretSave(pluginId: string, name: string, key: string) {
  installation.value.writeSecret(pluginId, name, key)
}

function onSecretDelete(pluginId: string, name: string) {
  installation.value.deleteSecret(pluginId, name)
}
</script>

<template>
  <section class="settings__group" :aria-label="words.ai">
    <h2 class="settings__heading">{{ words.ai }}</h2>

    <div
      v-for="plugin in waitingPlugins"
      :key="`asking-${plugin.id}`"
      class="settings__asking"
      role="alert"
    >
      <span>{{ words.asking(plugin.name) }}</span>
      <span class="settings__asking-answers">
        <Button size="small" @click="onConsentChange(plugin.id, true)">{{ words.allow }}</Button>
        <Button variant="outline" size="small" @click="onConsentChange(plugin.id, false)">
          {{ words.refuse }}
        </Button>
      </span>
    </div>

    <AgentChoice
      at="ai-default"
      :agent-name="words.defaultAgent"
      :agent-detail="words.defaultAgentDetail"
      :model-name="words.defaultModel"
      :model-detail="words.defaultModelDetail"
      :agents="agentChoices"
      :models="modelChoices"
      :choice="defaults"
      @choose="onDefaultChoose"
    />

    <SettingRow
      v-for="plugin in askingPlugins"
      :key="`consent-${plugin.id}`"
      v-slot="{ labelledBy }"
      :at="`ai-consent-${plugin.id}`"
      :name="`${plugin.name}: ${words.consentAllowed}`"
      :detail="words.consentDetail[plugin.consent]"
    >
      <Switch
        :model-value="plugin.consent === 'allowed'"
        :aria-labelledby="labelledBy"
        @update:model-value="(isAllowed) => onConsentChange(plugin.id, isAllowed)"
      />
    </SettingRow>
  </section>

  <section v-if="pluginsWithSecrets.length > 0" class="settings__group" :aria-label="words.keys">
    <h2 class="settings__heading">{{ words.keys }}</h2>

    <template v-for="plugin in pluginsWithSecrets" :key="`keys-${plugin.id}`">
      <SecretField
        v-for="secret in getShownSecrets(plugin)"
        :key="`${plugin.id}-${secret.name}`"
        :at="`key-${plugin.id}-${secret.name}`"
        :name="`${plugin.name}: ${secret.title}`"
        :is-stored="plugin.storedSecrets.includes(secret.name)"
        :words="words"
        @save="(key) => onSecretSave(plugin.id, secret.name, key)"
        @delete="onSecretDelete(plugin.id, secret.name)"
      />
    </template>
  </section>
</template>

<style scoped>
.settings__group {
  max-inline-size: var(--settings-measure, 54rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  margin-inline: auto;
}

.settings__heading {
  margin: 0 0 var(--settings-near, 0.375rem);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* A plugin waiting for an answer stands above the rows, with the two answers. */
.settings__asking {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near, 0.375rem);
  margin-block-end: var(--settings-near, 0.375rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke, 1px) solid var(--numen-accent, currentColor);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
}

.settings__asking-answers {
  display: flex;
  gap: var(--settings-near, 0.375rem);
}
</style>
