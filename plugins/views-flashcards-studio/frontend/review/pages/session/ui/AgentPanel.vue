<script setup lang="ts">
/**
 * The panel the card conversation is shown in: the conversation view a plugin declares, mounted into it.
 * Where no plugin declares one, the panel stays empty and the review continues without it.
 */
import { inject, useTemplateRef } from 'vue'

import { MOUNT_VIEW, mountNothing } from '../../../shared/views'
import { useMountedConversation } from '../model/mountedConversation'
import type { AgentPanelState } from '../model/panel'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ panel: AgentPanelState }>()

/* --------------------------------- State ---------------------------------- */
const element = useTemplateRef<HTMLElement>('element')

useMountedConversation({
  element,
  context: () => props.panel.context.value,
  open: () => props.panel.open.value,
  mountView: inject(MOUNT_VIEW, mountNothing),
})
</script>

<template>
  <section ref="element" class="asking" aria-label="Ask about this card" />
</template>

<style scoped>
/* The panel replaces the card in the same position and looks the same, so it
   takes the card's ground. */
.asking {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-inline-size: 0;
  min-block-size: 0;
  padding: var(--numen-inset-wide);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
}
</style>
