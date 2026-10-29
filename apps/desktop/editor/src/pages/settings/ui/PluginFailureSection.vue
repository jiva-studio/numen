<script setup lang="ts">
/**
 * The plugins the host suspended after they failed to start: the reason, the
 * time, and the two actions available to the user.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { SettingsTabState } from '../types'
import { write } from '@/entities/settings'
import type { PluginInfo } from '@/entities/settings'
import { formatFailedAt, getFailedPlugins } from '../lib/failures'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const failed = computed<readonly PluginInfo[]>(() =>
  getFailedPlugins(installation.value.plugins?.value ?? []),
)

/* -------------------------------- Handlers -------------------------------- */
function onRetry(pluginId: string) {
  installation.value.retryPlugin(pluginId)
}

function onDisable(pluginId: string) {
  installation.value.write([{ at: ['plugins', pluginId, 'enabled'], value: write(false) }])
}
</script>

<template>
  <section v-if="failed.length > 0" class="settings__group" :aria-label="words.pluginsStopped">
    <h2 class="settings__heading">{{ words.pluginsStopped }}</h2>

    <div
      v-for="plugin in failed"
      :key="plugin.id"
      class="settings__stopped"
      role="group"
      :aria-label="plugin.name"
    >
      <div class="settings__stopped-why">
        <span class="settings__stopped-name">{{ plugin.name }}</span>
        <span class="settings__stopped-reason">{{ plugin.failure?.reason }}</span>
        <span v-if="plugin.failure" class="settings__stopped-when">
          {{ words.pluginStoppedAt(formatFailedAt(plugin.failure.failedAt)) }}
        </span>
      </div>
      <span class="settings__stopped-answers">
        <Button
          size="small"
          :aria-label="words.pluginRetryFor(plugin.name)"
          @click="onRetry(plugin.id)"
        >
          {{ words.pluginRetry }}
        </Button>
        <Button
          variant="outline"
          size="small"
          :aria-label="words.pluginDisableFor(plugin.name)"
          @click="onDisable(plugin.id)"
        >
          {{ words.pluginDisable }}
        </Button>
      </span>
    </div>
  </section>
</template>

<style scoped>
.settings__group {
  max-inline-size: var(--settings-measure);
  margin-block-end: var(--settings-apart);
  margin-inline: auto;
}

.settings__heading {
  margin: 0 0 var(--settings-near);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.settings__stopped {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near);
  margin-block-end: var(--settings-near);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius-small);
  color: var(--numen-ink);
}

.settings__stopped-why {
  display: flex;
  flex-direction: column;
  min-inline-size: 0;
  gap: 0.125rem;
}

.settings__stopped-name {
  font-weight: 600;
}

.settings__stopped-reason {
  overflow-wrap: anywhere;
}

.settings__stopped-when {
  color: var(--numen-hushed);
}

.settings__stopped-answers {
  display: flex;
  gap: var(--settings-near);
}
</style>
