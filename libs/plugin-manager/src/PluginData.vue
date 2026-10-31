<script setup lang="ts">
/**
 * The folder a plugin keeps in the shown vault: how many files it holds, and the deletion, which
 * the person confirms before it is asked for.
 */
import { ref } from 'vue'
import { Button } from '@numen/ui'
import DetailRow from './DetailRow.vue'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  pluginId: string
  pluginName: string
  /** How many files the plugin keeps in the shown vault; zero hides the delete action. */
  fileCount: number
  /** Why the last attempt to delete failed; empty otherwise. */
  problem: string
}>()

const emit = defineEmits<{
  (e: 'delete-data'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isAsking = ref(false)

/* -------------------------------- Handlers -------------------------------- */
function onDelete(): void {
  isAsking.value = true
}

function onKeep(): void {
  isAsking.value = false
}

function onConfirm(): void {
  isAsking.value = false
  emit('delete-data')
}
</script>

<template>
  <div class="plugin-data">
    <div v-if="props.fileCount > 0" class="plugin-data__card">
      <DetailRow
        :at="`data-${props.pluginId}`"
        :name="words.pluginData"
        :detail="words.pluginDataDetail(props.fileCount)"
      >
        <Button variant="danger" size="small" @click="onDelete">
          {{ words.pluginDataDelete }}
        </Button>
      </DetailRow>
    </div>
    <div
      v-if="isAsking"
      class="plugin-data__asking"
      role="alert"
      :aria-label="words.pluginDataDelete"
    >
      <span>{{ words.pluginDataAsking(props.pluginName, props.fileCount) }}</span>
      <span class="plugin-data__answers">
        <Button variant="danger" size="small" @click="onConfirm">
          {{ words.pluginDataConfirm }}
        </Button>
        <Button variant="outline" size="small" @click="onKeep">
          {{ words.pluginDataKeep }}
        </Button>
      </span>
    </div>
    <p v-if="props.problem" class="plugin-data__problem" role="alert">{{ props.problem }}</p>
  </div>
</template>

<style scoped>
.plugin-data__card {
  padding-inline: 1rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-data__asking {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--settings-near, 0.375rem);
  margin-block-start: var(--settings-near, 0.375rem);
  padding: 0.5rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-alarm);
  border-radius: var(--numen-radius-small);
  color: var(--numen-ink);
}

.plugin-data__answers {
  display: flex;
  gap: var(--settings-near, 0.375rem);
}

.plugin-data__problem {
  margin: var(--settings-near, 0.375rem) 0 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-1);
}
</style>
