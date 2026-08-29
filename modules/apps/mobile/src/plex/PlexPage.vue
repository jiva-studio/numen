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
import { byHolding, Plex, type PlexNeighbourhood, type PlexRelatedSeat } from '@numen/ui'
import { reach, type Reached } from '../core'
import { follow } from './following'
import { asPlex } from './picture'
import { CREATABLE, ROLES, SEEDED } from './seats'

const core = ref<Reached | null>(null)
const picture = ref<PlexNeighbourhood | null>(null)
const at = ref(SEEDED)
const trouble = ref('')

/** A finger has no hover, so a node is reached out of by resting on it. */
const reaching = byHolding()

async function draw(path: string) {
  if (!core.value) return
  const said = await core.value.vault.neighbourhood({ path })
  at.value = path
  picture.value = asPlex(said)
}

async function made(from: string, seat: PlexRelatedSeat) {
  if (!core.value) return
  const title = window.prompt(`A new ${seat}`)
  if (!title?.trim()) return
  const created = await core.value.vault.create({ title: title.trim(), folder: '' })
  if (!created.path) {
    trouble.value = `nothing was made: ${JSON.stringify(created.refusal)}`
    return
  }
  await joined(from, created.path, seat)
}

async function joined(from: string, to: string, seat: PlexRelatedSeat) {
  if (!core.value) return
  const said = await core.value.vault.join({ path: from, link: { to, role: ROLES[seat] } })
  if (said.refusal) {
    trouble.value = `nothing was written: ${JSON.stringify(said.refusal)}`
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
    trouble.value = String(why)
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
        @activate="(node: string) => void draw(node)"
        @create="(from: string, seat: PlexRelatedSeat) => void made(from, seat)"
        @link="(from: string, to: string, seat: PlexRelatedSeat) => void joined(from, to, seat)"
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
</template>

<style scoped>
/* The picture is the page: it fills what the content leaves and scrolls
   nothing, because panning it is the gesture. */
.plex {
  width: 100%;
  height: 100%;
}
</style>
