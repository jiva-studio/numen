<script setup lang="ts">
/**
 * The recovery kit of the identity as one row: what the kit is, and the action that saves a new kit
 * as a PDF at the right of the row. The action is in the row, not in the header of the card, because
 * the row is about one object. The row says why a kit was not saved.
 */
import { Download } from '@lucide/vue'
import { Button, Spinner, Tooltip, useTooltip } from '@numen/ui'
import { computed } from 'vue'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'download'): void
}>()

/* --------------------------------- State ---------------------------------- */
const tip = useTooltip()
const label = computed(() => (props.isBusy ? words.kitDownloading : words.kitDialogTitle))
</script>

<template>
  <SettingRow
    at="identity-kit"
    :name="words.kitTitle"
    :detail="words.kitDetail"
    :details="words.kitDetails"
  >
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="recovery-kit__download"
      :aria-label="label"
      :aria-describedby="tip.describedBy.value"
      @mouseenter="tip.open"
      @mouseleave="tip.close"
      @focus="tip.open"
      @blur="tip.close"
      :disabled="isBusy"
      @click="emit('download')"
    >
      <Spinner v-if="isBusy" />
      <Download v-else :size="16" aria-hidden="true" />
      <Teleport to="body">
        <Tooltip v-if="tip.at.value" :id="tip.id" :at="tip.at.value">{{ label }}</Tooltip>
      </Teleport>
    </Button>
  </SettingRow>
  <p v-if="problem !== ''" role="alert" class="recovery-kit__problem">{{ problem }}</p>
</template>

<style scoped>
.recovery-kit__download {
  justify-self: end;
}

.recovery-kit__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
