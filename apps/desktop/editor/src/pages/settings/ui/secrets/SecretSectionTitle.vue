<script setup lang="ts">
/**
 * The header of the section of a group: the title and description of its service, the button that
 * adds a secret and, for a group the person made, a menu that renames and deletes the group. A
 * group is renamed in place; a name that is too long or is the name of another group is refused
 * with an alert and the field stays open.
 */
import { computed, onMounted, ref, useTemplateRef } from 'vue'
import { Plus } from '@lucide/vue'
import { Button, type MenuItem } from '@numen/ui'
import {
  MAX_NAME_LENGTH,
  getServiceInfo,
  isGroupNameTaken,
  isNameTooLong,
  trimName,
} from '@/entities/settings'
import { runAfterRender } from '../../model/afterRender'
import { WORDS as words } from '../../words'
import SecretInput from './SecretInput.vue'
import SecretMenu from './SecretMenu.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  group: string
  groups: readonly string[]
  headingId: string
  isAsked: boolean
  isNew: boolean
}>()

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'rename', name: string): void
  (e: 'delete-group', opener: HTMLElement): void
}>()

defineExpose({
  focusAdd: () => (addButton.value?.$el as HTMLElement | undefined)?.focus(),
})

/* --------------------------------- State ---------------------------------- */
const items: readonly MenuItem[] = [
  { id: 'rename', text: words.secretGroupRename },
  { id: 'delete', text: words.secretGroupDelete },
]
const isRenaming = ref(false)
const draftName = ref('')
const nameProblem = ref<'' | 'long' | 'taken'>('')
const info = computed(() => getServiceInfo(props.group))
const nameInput = useTemplateRef<InstanceType<typeof SecretInput>>('nameInput')
const addButton = useTemplateRef<InstanceType<typeof Button>>('addButton')

onMounted(() => {
  if (props.isNew) onStartRename()
})

/* -------------------------------- Handlers -------------------------------- */
function onStartRename(): void {
  draftName.value = props.group
  nameProblem.value = ''
  isRenaming.value = true
  runAfterRender(() => {
    nameInput.value?.focus()
    nameInput.value?.select()
  })
}

function onCommitRename(): void {
  if (!isRenaming.value) return
  const name = trimName(draftName.value)
  nameProblem.value = getNameProblem(name)
  if (nameProblem.value !== '') return
  isRenaming.value = false
  if (name !== '' && name !== props.group) emit('rename', name)
}

function onCancelRename(): void {
  isRenaming.value = false
  nameProblem.value = ''
}

function onChoose(id: string, opener: HTMLElement): void {
  if (id === 'rename') onStartRename()
  else emit('delete-group', opener)
}

/* -------------------------------- Helpers --------------------------------- */
function getNameProblem(name: string): '' | 'long' | 'taken' {
  if (isNameTooLong(name)) return 'long'
  return isGroupNameTaken(props.groups, props.group, name) ? 'taken' : ''
}

function getNameProblemText(): string {
  return nameProblem.value === 'long'
    ? words.secretGroupTooLong(MAX_NAME_LENGTH)
    : words.secretGroupTaken(trimName(draftName.value))
}
</script>

<template>
  <div class="secret-title">
    <div class="secret-title__text">
      <SecretInput
        v-if="isRenaming"
        ref="nameInput"
        v-model="draftName"
        type="text"
        :aria-label="words.secretGroupRenameName"
        :aria-invalid="nameProblem !== ''"
        @keydown.enter="onCommitRename"
        @keydown.esc="onCancelRename"
        @change="onCommitRename"
        @blur="onCommitRename"
      />
      <p v-if="nameProblem !== ''" role="alert" class="secret-title__refusal">
        {{ getNameProblemText() }}
      </p>
      <h2 v-show="!isRenaming" :id="headingId" class="secret-title__name">{{ info.title }}</h2>
      <p v-if="info.description !== '' && !isRenaming" class="secret-title__about">
        {{ info.description }}
      </p>
    </div>
    <Button v-if="!isRenaming" ref="addButton" variant="outline" @click="emit('add')">
      <Plus :size="14" aria-hidden="true" />
      {{ words.secretAdd }}
    </Button>
    <SecretMenu
      v-if="!isRenaming && !isAsked"
      :label="words.secretGroupActions"
      :items="items"
      @choose="onChoose"
    />
  </div>
</template>

<style scoped>
.secret-title {
  --gap: 0.25rem;
  --pad-end: 0.375rem;
  display: flex;
  align-items: center;
  gap: var(--gap);
  padding-block-end: var(--pad-end);
}

.secret-title__text {
  --margin-end: 0.5rem;
  --gap: 0.125rem;
  margin-inline-end: var(--margin-end);
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--gap);
  min-inline-size: 0;
}

.secret-title__about,
.secret-title__refusal {
  margin: 0;
  font-size: var(--numen-text-1);
}

.secret-title__about {
  color: var(--numen-hushed);
}

.secret-title__refusal {
  color: var(--numen-alarm);
}

.secret-title__name {
  margin: 0;
  color: var(--numen-ink);
  font-size: var(--numen-title-size);
  font-weight: 500;
}
</style>
