<script setup lang="ts">
/**
 * The head of a setting of rows: its title, and the button that adds a row where the setting takes
 * more than one, and the switch "Discover automatically" where the setting declares discover.
 */
import { Plus } from '@lucide/vue'
import { Button, Switch } from '@numen/ui'
import SettingRow from '../../setting-row/SettingRow.vue'
import { WORDS as words } from '../../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  canAdd: boolean
  isAddDisabled: boolean
  isDiscoverShown: boolean
  isDiscovering: boolean
}>()

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'discover', isOn: boolean): void
}>()
</script>

<template>
  <SettingRow :at="props.at" :name="props.name" detail="">
    <Button
      v-if="props.canAdd"
      variant="outline"
      :disabled="props.isAddDisabled"
      @click="emit('add')"
    >
      <Plus :size="14" aria-hidden="true" />
      {{ words.connectionAdd }}
    </Button>
  </SettingRow>

  <SettingRow
    v-if="props.isDiscoverShown"
    v-slot="{ labelledBy }"
    :at="`${props.at}-discover`"
    :name="words.connectionRowsDiscover"
    detail=""
  >
    <Switch
      :model-value="props.isDiscovering"
      :aria-labelledby="labelledBy"
      @update:model-value="(isOn) => emit('discover', isOn)"
    />
  </SettingRow>
</template>
