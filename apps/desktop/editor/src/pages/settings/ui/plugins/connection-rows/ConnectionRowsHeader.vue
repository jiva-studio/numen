<script setup lang="ts">
/**
 * The head of a setting of rows: its title, and the button that adds a row where the setting takes
 * more than one.
 */
import { Plus } from '@lucide/vue'
import { Button } from '@numen/ui'
import SettingRow from '../../setting-row/SettingRow.vue'
import { WORDS as words } from '../../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  at: string
  name: string
  canAdd: boolean
  isAddDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'add'): void
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
</template>
