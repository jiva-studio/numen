<script setup lang="ts">
/**
 * One secret of a group as a row: its name as an input where the group holds more than one secret, the value
 * (dots as the placeholder while one is stored), the plugins that may read it, and the menu to move and delete it.
 */
import { computed, ref, watch } from 'vue'
import type { MenuItem } from '@numen/ui'
import { trimName, type Secret } from '@/entities/settings'
import type { Declaration } from '../../lib/secretSections'
import { WORDS as words } from '../../words'
import SecretMenu from './SecretMenu.vue'
import SecretNameInput from './SecretNameInput.vue'
import SecretPermissions from './SecretPermissions.vue'
import SecretRowFrame from './SecretRowFrame.vue'
import SecretValueInput from './SecretValueInput.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  secret: Secret
  isFirst: boolean
  isLast: boolean
  canMove: boolean
  hasName: boolean
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
const items = computed<readonly MenuItem[]>(() => [
  ...(props.canMove
    ? [
        { id: 'up', text: words.secretMoveUp, isDisabled: props.isFirst },
        { id: 'down', text: words.secretMoveDown, isDisabled: props.isLast },
      ]
    : []),
  { id: 'delete', text: words.secretDeleteItem },
])

watch(
  () => props.secret.name,
  (name) => {
    draftName.value = name
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onCommitRename(): void {
  const name = trimName(draftName.value)
  if (name !== '' && name !== props.secret.name) emit('rename', name)
  else draftName.value = props.secret.name
}

function onChoose(id: string, opener: HTMLElement): void {
  if (id === 'up') emit('move-up')
  else if (id === 'down') emit('move-down')
  else if (id === 'delete') emit('delete', opener)
}
</script>

<template>
  <SecretRowFrame :has-name="hasName" :label="secret.name">
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
        :has-value="secret.hasValue === true"
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
      <SecretMenu :label="words.secretActions" :items="items" @choose="onChoose" />
    </template>
  </SecretRowFrame>
</template>
