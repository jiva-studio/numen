<script setup lang="ts">
import type { PluginSettingOption } from '@/entities/settings'
import type { ProcessHost } from '@numen/plugin-loader'
import { useModelProviders } from '../../model/useModelProviders'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  pluginId: string
  processes: ProcessHost
  options: readonly Pick<PluginSettingOption, 'value' | 'title'>[]
}>()

/* --------------------------------- State ---------------------------------- */
const { status, providers } = useModelProviders(
  () => props.pluginId,
  () => props.processes,
  () => props.options,
)
</script>

<template>
  <SettingRow
    v-if="providers.length || status"
    at="model-providers"
    :name="words.executionProvider"
    detail=""
  >
    <ul v-if="providers.length">
      <li v-for="entry in providers" :key="entry.key">
        {{ entry.stage }} · {{ entry.provider }}
      </li>
    </ul>
    <span v-else>{{ status }}</span>
  </SettingRow>
</template>
