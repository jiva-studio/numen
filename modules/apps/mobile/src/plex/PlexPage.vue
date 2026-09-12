<script setup lang="ts">
/**
 * The vault as a picture, and what a gesture over it does to the vault.
 *
 * The picture is the whole page. A tap travels to the node tapped; holding a
 * node reaches out of it, and letting go in a seat makes a note there or joins
 * the node it was let go on.
 */
import { onMounted, ref } from 'vue'
import { IonContent, IonPage, IonProgressBar, IonToast } from '@ionic/vue'
import {
  byDoubleTap,
  byHolding,
  Plex,
  type PlexNeighbourhood,
  type PlexRelatedSeat,
} from '@numen/ui'
import { formatErrorCodeMessage, formatErrorMessage } from '@numen/wire'
import NoteSheet from '../note/NoteSheet.vue'
import { reach, type Core } from '../core'
import { follow } from './following'
import { asPlex } from './picture'
import { CREATABLE, isCreatable, ROLES, SEEDED } from './seats'

const core = ref<Core | null>(null)
const picture = ref<PlexNeighbourhood | null>(null)
const at = ref(SEEDED)
const trouble = ref('')

/** A finger has no hover, so a node is reached out of by resting on it. */
const reaching = byHolding()

/** And asked for on its own by tapping it twice. */
const showing = byDoubleTap()

/** The note being written, and nothing while none is. */
const writing = ref<string | null>(null)

async function draw(path: string) {
  if (!core.value) return
  const said = await core.value.notes.getNeighbourhood({ path })
  at.value = path
  picture.value = asPlex(said)
}

async function createRelatedNote(from: string, seat: PlexRelatedSeat) {
  if (!core.value) return
  const title = window.prompt(`A new ${seat}`)
  if (!title?.trim()) return
  const created = await core.value.notes.createNote({ title: title.trim(), path: '' })
  if (!created.path) {
    trouble.value = formatErrorCodeMessage(created.refusal)
    return
  }
  await linkNotes(from, created.path, seat)
}

async function linkNotes(from: string, to: string, seat: PlexRelatedSeat) {
  if (!core.value) return
  // The picture is only ever asked for a seat it offers, and it offers no
  // sibling: no link writes one.
  if (!isCreatable(seat)) return
  const said = await core.value.notes.writeLink({ path: from, link: { to, role: ROLES[seat] } })
  if (said.refusal) {
    trouble.value = formatErrorCodeMessage(said.refusal)
    return
  }
  await draw(at.value)
}

onMounted(async () => {
  try {
    core.value = await reach()
    await draw(SEEDED)
    // The vault is still being read; the picture is drawn again as it lands.
    follow(core.value, () => void draw(at.value))
  } catch (why) {
    trouble.value = formatErrorMessage(why)
  }
})
</script>

<template>
  <IonPage>
    <IonContent :fullscreen="true" :scroll-y="false">
      <IonProgressBar v-if="!picture && !trouble" type="indeterminate" />
      <Plex
        v-if="picture"
        class="plex"
        data-testid="plex"
        :neighbourhood="picture"
        :creatable="CREATABLE"
        :reaching="reaching"
        :showing="showing"
        @activate="(node: string) => void draw(node)"
        @show="(node: string) => (writing = node)"
        @create="(from: string, seat: PlexRelatedSeat) => void createRelatedNote(from, seat)"
        @link="(from: string, to: string, seat: PlexRelatedSeat) => void linkNotes(from, to, seat)"
      />
      <IonToast
        :is-open="!!trouble"
        :message="trouble"
        color="danger"
        :duration="6000"
        data-testid="trouble"
        @did-dismiss="trouble = ''"
      />
    </IonContent>
  </IonPage>
  <!-- The note is put under the body: an editor keeps whichever root it was
       built in, and every wrapper of the framework has one of its own. -->
  <Teleport to="body">
    <NoteSheet
      v-if="core && writing"
      :core="core"
      :path="writing"
      @close="writing = null; void draw(at)"
      @trouble="(said: string) => (trouble = said)"
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
