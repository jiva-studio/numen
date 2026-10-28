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
  isMulti?: boolean
  words: {
    readonly pathNone: string
    readonly pathChoose: string
    readonly pathAdd?: string
    readonly pathClear: string
  }
}>()

const emit = defineEmits<{
  choose: []
  clear: [targetPath?: string]
}>()

/* --------------------------------- State ---------------------------------- */
const paths = computed(() => parsePaths(props.path))
const hasMultipleRows = computed(() => isMultiRowLayout(props.isMulti, paths.value))
const chooseButtonLabel = computed(() => getChooseButtonLabel(props.isMulti, props.words))

/* -------------------------------- Handlers -------------------------------- */
function onChoose() {
  emit('choose')
}

function onClear(targetPath?: string) {
  emit('clear', targetPath)
}

/* -------------------------------- Helpers --------------------------------- */
function parsePaths(rawPaths: string): string[] {
  return rawPaths
    .split('\n')
    .map((one) => one.trim())
    .filter((one) => one !== '')
}

function isMultiRowLayout(isMulti: boolean | undefined, items: readonly string[]): boolean {
  return Boolean(isMulti) || items.length > 1
}

function getChooseButtonLabel(
  isMulti: boolean | undefined,
  words: { readonly pathChoose: string; readonly pathAdd?: string },
): string {
  if (isMulti) {
    return words.pathAdd ?? words.pathChoose
  }
  return words.pathChoose
}

function formatPathLabel(rawPath: string): string {
  const parts = rawPath.split(/[\\/]/).filter((one) => one !== '')
  return parts.length > 2 ? `…/${parts.slice(-2).join('/')}` : rawPath
}
</script>

<template>
  <span class="settings__path" role="group" :aria-labelledby="labelledBy">
    <template v-if="!hasMultipleRows">
      <span class="settings__path-line">
        <span
          v-if="paths[0]"
          class="settings__path-label"
          :title="paths[0]"
          data-testid="setting-path"
        >
          {{ formatPathLabel(paths[0]) }}
        </span>
        <Button variant="outline" size="small" class="settings__path-choose" @click="onChoose">
          {{ chooseButtonLabel }}
        </Button>
        <Button v-if="paths[0]" variant="ghost" size="small" @click="onClear(paths[0])">
          {{ words.pathClear }}
        </Button>
      </span>
    </template>
    <template v-else>
      <span v-for="item in paths" :key="item" class="settings__path-line">
        <span class="settings__path-label" :title="item" data-testid="setting-path">
          {{ formatPathLabel(item) }}
        </span>
        <Button variant="ghost" size="small" @click="onClear(item)">
          {{ words.pathClear }}
        </Button>
      </span>
      <span class="settings__path-line">
        <Button variant="outline" size="small" class="settings__path-choose" @click="onChoose">
          {{ chooseButtonLabel }}
        </Button>
      </span>
    </template>
    <span v-if="problem !== ''" class="settings__path-problem" role="alert">{{ problem }}</span>
  </span>
</template>

<style scoped>
.settings__path {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.375rem;
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

.settings__path-agent-title {
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
  font-weight: 500;
  white-space: nowrap;
}

.settings__path-line--discovered .settings__path-label {
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono, monospace);
  font-size: var(--numen-text-1);
}

.settings__path-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-inline-size: 18rem;
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
}

.settings__path-choose {
  background-color: var(--numen-field-bg, var(--numen-surface));
  border-color: var(--numen-field-rule, var(--numen-rule));
}

.settings__path-problem {
  color: var(--numen-alarm, #d14);
  font-size: var(--numen-text-1);
}
</style>
