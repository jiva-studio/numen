<script setup lang="ts">
/**
 * The step of the palette that asks for several items: a title over the list that
 * checks them. A toggle or a create asks the caller and leaves the list open.
 *
 * `data-palette` names the parts: `title`, and `field`, the line the text is typed in.
 */
import { computed, ref } from 'vue'
import { SearchableList, type ListItem } from '@numen/ui'
import type { PaletteMultiProps } from './PaletteMulti.types'

const props = defineProps<{ multi: PaletteMultiProps }>()

defineEmits<{
  /** An item was asked to be checked or unchecked. */
  (event: 'toggle', id: string): void
  /** The row that creates an item was chosen, with the text as typed. */
  (event: 'create', text: string): void
  /** Requests that the palette close. */
  (event: 'dismiss'): void
}>()

const holder = ref<HTMLElement | null>(null)

const items = computed<readonly ListItem[]>(() =>
  props.multi.items.map((item) => ({ id: item.id, text: item.title, state: item.state })),
)
</script>

<template>
  <div ref="holder" class="palette__multi">
    <p class="palette__multi-title caps-numen text-small text-hushed" data-palette="title">
      {{ multi.title }}
    </p>
    <SearchableList
      multiple
      has-action
      :items="items"
      :recent="multi.recent"
      :label="multi.title"
      :placeholder="multi.placeholder"
      :input-attrs="{ 'data-palette': 'field' }"
      :from="holder"
      @toggle="$emit('toggle', $event)"
      @act="$emit('create', $event)"
      @close="$emit('dismiss')"
    >
      <template #action="{ query }">{{ multi.getCreateTitle(query) }}</template>
    </SearchableList>
  </div>
</template>

<style scoped>
.palette__multi-title {
  --pad: 0.5rem 0.75rem 0;

  margin: 0;
  padding: var(--pad);
}

/* The panel is the frame; the list draws none of its own. */
.palette__multi :deep(.searchable-list) {
  background: transparent;
  border: 0;
  box-shadow: none;
}
</style>
