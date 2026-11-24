<script setup lang="ts">
/**
 * The "..." action of the card header and the menu it opens: export the account data, and delete the cloud
 * data, which is marked as dangerous. Choosing an item emits it; nothing is done here.
 */
import { computed, defineComponent, h, markRaw, ref, useTemplateRef } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { CardAction, Menu, measureRect, type MenuItem, type Position } from '@numen/ui'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const emit = defineEmits<{
  (event: 'export'): void
  (event: 'delete'): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof CardAction>>('trigger')
const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

const DANGER = markRaw(
  defineComponent({
    name: 'DeleteCloudDataItem',
    setup: () => () => h('span', { class: 'subscription__danger' }, words.deleteCloudData),
  }),
)

const ITEMS: readonly MenuItem[] = [
  { id: 'export', text: words.exportAccountData, group: 'safe' },
  { id: 'delete', text: words.deleteCloudData, group: 'danger', component: DANGER },
]

const anchor = ref<{ at: Position; wide: number } | null>(null)
const items = computed(() => ITEMS)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(): void {
  if (anchor.value !== null) {
    onClose()
    return
  }
  const box = measureRect(getTrigger())
  if (box !== undefined) anchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  onClose()
  if (id === 'export') emit('export')
  else if (id === 'delete') emit('delete')
}
</script>

<template>
  <CardAction
    ref="trigger"
    is-primary
    is-icon-only
    :label="words.moreActions"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    @click="onToggle"
  >
    <Ellipsis :size="14" aria-hidden="true" />
  </CardAction>
  <Menu
    v-if="anchor"
    :items="items"
    :at="anchor.at"
    :asking="anchor.wide"
    :from="getTrigger()"
    :name="words.moreActions"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="onClose"
  />
</template>
