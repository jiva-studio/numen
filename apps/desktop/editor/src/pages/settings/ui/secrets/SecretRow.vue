<script setup lang="ts">
/**
 * One secret of a group as a row of four columns: its name as an input, the value
 * (dots as the placeholder while one is stored), the plugins that may read it, and the controls to move and delete.
 */
import { ref, watch } from 'vue'
import { ArrowDown, ArrowUp, Trash2 } from '@lucide/vue'
import { Button } from '@numen/ui'
import type { Secret } from '@/entities/settings'
import type { Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SecretNameInput from './SecretNameInput.vue'
import SecretPermissions from './SecretPermissions.vue'
import SecretRowFrame from './SecretRowFrame.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret: Secret
  isFirst: boolean
  isLast: boolean
  /** Whether the group holds more than one secret, so that the order can change. */
  canMove: boolean
  declarations: readonly Declaration[]
}>()

const emit = defineEmits<{
  (e: 'move-up'): void
  (e: 'move-down'): void
  (e: 'rename', name: string): void
  (e: 'write', value: string): void
  (e: 'delete', opener: HTMLElement): void
  (e: 'toggle', declaration: Declaration, isAllowed: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const draftName = ref(props.secret.name)

watch(
  () => props.secret.name,
  (name) => {
    draftName.value = name
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onCommitRename(): void {
  const name = draftName.value.trim()
  if (name !== '' && name !== props.secret.name) emit('rename', name)
  else draftName.value = props.secret.name
}

function onDelete(event: MouseEvent): void {
  if (event.currentTarget instanceof HTMLElement) emit('delete', event.currentTarget)
}
</script>

<template>
  <SecretRowFrame>
    <template #name>
      <SecretNameInput
        v-model="draftName"
        @commit="onCommitRename"
        @cancel="draftName = secret.name"
      />
    </template>

    <template #value>
      <SecretValueInput
        :label="words.secretValue(secret.name)"
        :placeholder="secret.hasValue ? words.secretMasked : words.keyPlaceholder"
        @commit="(value) => emit('write', value)"
      />
    </template>

    <template #permissions>
      <SecretPermissions
        :secret="secret"
        :declarations="declarations"
        @toggle="(declaration, isAllowed) => emit('toggle', declaration, isAllowed)"
      />
    </template>

    <template #actions>
      <Button
        v-if="canMove"
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretMoveUp"
        :title="words.secretMoveUp"
        :disabled="isFirst"
        @click="emit('move-up')"
      >
        <ArrowUp :size="14" aria-hidden="true" />
      </Button>
      <Button
        v-if="canMove"
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretMoveDown"
        :title="words.secretMoveDown"
        :disabled="isLast"
        @click="emit('move-down')"
      >
        <ArrowDown :size="14" aria-hidden="true" />
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        :aria-label="words.secretDelete"
        :title="words.secretDelete"
        @click="onDelete"
      >
        <Trash2 :size="14" aria-hidden="true" />
      </Button>
    </template>
  </SecretRowFrame>
</template>
