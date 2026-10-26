<script setup lang="ts">
/**
 * The path a plugin's setting of type file or folder holds: its last parts label and the whole
 * path on hover, the machine's own picker to choose another, and the path cleared. Why the setting
 * did not take the path chosen is said under it.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  path: string
  /** Why the setting did not take the path last chosen; empty where it did. */
  problem: string
  labelledBy: string
  words: {
    readonly pathNone: string
    readonly pathChoose: string
    readonly pathClear: string
  }
}>()

const emit = defineEmits<{
  choose: []
  clear: []
}>()

/* --------------------------------- State ---------------------------------- */
/** The folder and the name at the end of the path. */
const label = computed(() => {
  if (props.path === '') return props.words.pathNone
  const parts = props.path.split(/[\\/]/).filter((one) => one !== '')
  return parts.length > 2 ? `…/${parts.slice(-2).join('/')}` : props.path
})

/* -------------------------------- Handlers -------------------------------- */
function onChoose() {
  emit('choose')
}

function onClear() {
  emit('clear')
}
</script>

<template>
  <span class="settings__path" role="group" :aria-labelledby="labelledBy">
    <span class="settings__path-line">
      <span
        class="settings__path-label"
        :class="{ 'settings__path-label--none': path === '' }"
        :title="path"
        data-testid="setting-path"
      >
        {{ label }}
      </span>
      <Button variant="outline" size="small" @click="onChoose">{{ words.pathChoose }}</Button>
      <Button v-if="path !== ''" variant="ghost" size="small" @click="onClear">
        {{ words.pathClear }}
      </Button>
    </span>
    <span v-if="problem !== ''" class="settings__path-problem" role="alert">{{ problem }}</span>
  </span>
</template>

<style scoped>
.settings__path {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.25rem;
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__path-line {
  display: flex;
  align-items: center;
  gap: var(--settings-near, 0.375rem);
  min-inline-size: 0;
  max-inline-size: 100%;
}

.settings__path-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-inline-size: 16rem;
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
}

.settings__path-label--none {
  color: var(--numen-hushed);
}

.settings__path-problem {
  color: var(--numen-alarm, #d14);
  font-size: var(--numen-text-1);
}
</style>
