<template>
  <div class="embedding-settings">
    <h3>Vector Embeddings Settings</h3>
    <div class="setting-item">
      <label for="provider-select">Provider:</label>
      <select id="provider-select" v-model="selectedProvider" @change="onProviderChanged">
        <option value="local">Local (On-device)</option>
        <option value="service">Remote Service</option>
      </select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    initialProvider?: string
  }>(),
  {
    initialProvider: 'local',
  },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'change-provider', provider: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const selectedProvider = ref(props.initialProvider)

/* -------------------------------- Handlers -------------------------------- */
function onProviderChanged(): void {
  emit('change-provider', selectedProvider.value)
}
</script>

<style scoped>
.embedding-settings {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.setting-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
</style>
