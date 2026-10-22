<script setup lang="ts">
/**
 * The vault as a picture, and what a gesture over it does to the vault.
 *
 * The picture is the whole page. A tap travels to the node tapped; holding a
 * node reaches out of it, and letting go in a relation makes a note there or joins
 * the node it was let go on.
 */
import { onMounted, ref } from 'vue'
import { IonContent, IonPage, IonProgressBar, IonToast } from '@ionic/vue'
import {
  byDoubleTap,
  byLongPress,
  Plex,
  type PlexNeighbourhood,
  type PlexRelation,
} from '@numen/ui'
import { formatErrorCodeMessage, formatErrorMessage } from '@numen/wire'
import NoteSheet from '../note/NoteSheet.vue'
import { startCore, type Core } from '../core'
import { watchVault } from './watchVault'
import { convertNeighbourhood } from './picture'
import { CREATABLE_RELATIONS, isCreatable, ROLE_BY_RELATION, SEED_NOTE_PATH } from './relations'

/* --------------------------------- State ---------------------------------- */
const core = ref<Core | null>(null)
const picture = ref<PlexNeighbourhood | null>(null)
const focusPath = ref(SEED_NOTE_PATH)
const errorMessage = ref('')

/** A finger has no hover, so a node is reached out of by resting on it. */
const reaching = byLongPress()

/** And asked for on its own by tapping it twice. */
const showing = byDoubleTap()

/** The note being written, and nothing while none is. */
const editingPath = ref<string | null>(null)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(async () => {
  try {
    core.value = await startCore()
    await loadPicture(SEED_NOTE_PATH)
    // The vault is still being read; the picture is drawn again as it lands.
    watchVault(core.value, () => void loadPicture(focusPath.value))
  } catch (why) {
    errorMessage.value = formatErrorMessage(why)
  }
})

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  editingPath.value = null
  void loadPicture(focusPath.value)
}

/* -------------------------------- Helpers --------------------------------- */
async function loadPicture(path: string) {
  if (!core.value) return
  const answer = await core.value.notes.getNeighbourhood({ path })
  focusPath.value = path
  picture.value = convertNeighbourhood(answer)
}

async function createRelatedNote(from: string, relation: PlexRelation) {
  if (!core.value) return
  const title = window.prompt(`A new ${relation}`)
  if (!title?.trim()) return
  const created = await core.value.notes.createNote({ title: title.trim(), path: '' })
  if (!created.path) {
    errorMessage.value = formatErrorCodeMessage(created.error)
    return
  }
  await linkNotes(from, created.path, relation)
}

async function linkNotes(from: string, to: string, relation: PlexRelation) {
  if (!core.value) return
  // The picture is only ever asked for a relation it offers, and it offers no
  // sibling: no link writes one.
  if (!isCreatable(relation)) return
  const answer = await core.value.notes.writeLink({
    path: from,
    link: { to, role: ROLE_BY_RELATION[relation] },
  })
  if (answer.error) {
    errorMessage.value = formatErrorCodeMessage(answer.error)
    return
  }
  await loadPicture(focusPath.value)
}
</script>

<template>
  <IonPage>
    <IonContent fullscreen :scroll-y="false">
      <IonProgressBar v-if="!picture && !errorMessage" type="indeterminate" />
      <Plex
        v-if="picture"
        class="plex"
        data-testid="plex"
        :neighbourhood="picture"
        :creatable="CREATABLE_RELATIONS"
        :reaching="reaching"
        :showing="showing"
        @activate="(node: string) => void loadPicture(node)"
        @show="(node: string) => (editingPath = node)"
        @create="(from: string, relation: PlexRelation) => void createRelatedNote(from, relation)"
        @link="
          (from: string, to: string, relation: PlexRelation) => void linkNotes(from, to, relation)
        "
      />
      <IonToast
        :is-open="!!errorMessage"
        :message="errorMessage"
        color="danger"
        :duration="6000"
        data-testid="error"
        @did-dismiss="errorMessage = ''"
      />
    </IonContent>
  </IonPage>
  <!-- The note is put under the body: an editor keeps whichever root it was
       built in, and every wrapper of the framework has one of its own. -->
  <Teleport to="body">
    <NoteSheet
      v-if="core && editingPath"
      :core="core"
      :path="editingPath"
      @close="onClose"
      @error="(message: string) => (errorMessage = message)"
    />
  </Teleport>
</template>

<style scoped>
/* The picture is the page: it fills what the content leaves and scrolls
   nothing. */
.plex {
  inline-size: 100%;
  block-size: 100%;
}
</style>
