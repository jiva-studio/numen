<script setup lang="ts">
/**
 * The whole of the spike: what the vault holds, and one note more.
 */
import { onMounted, ref } from 'vue'
import {
  IonApp,
  IonButton,
  IonContent,
  IonHeader,
  IonInput,
  IonItem,
  IonLabel,
  IonList,
  IonNote,
  IonTitle,
  IonToolbar,
} from '@ionic/vue'
import { reach, type Reached } from './core'

const core = ref<Reached | null>(null)
const entries = ref<{ path: string; name: string; folder: boolean }[]>([])
const trouble = ref('')
const title = ref('')

async function list() {
  if (!core.value) return
  const said = await core.value.vault.list({ folder: '' })
  entries.value = said.entries.map((e) => ({ path: e.path, name: e.name, folder: e.folder }))
}

async function make() {
  if (!core.value || !title.value.trim()) return
  const said = await core.value.vault.create({ title: title.value.trim(), folder: '' })
  if (said.refusal) {
    trouble.value = `refused: ${JSON.stringify(said.refusal)}`
    return
  }
  title.value = ''
  await list()
}

onMounted(async () => {
  try {
    core.value = await reach()
    await list()
  } catch (why) {
    trouble.value = String(why)
  }
})
</script>

<template>
  <IonApp>
    <IonHeader>
      <IonToolbar>
        <IonTitle>numen</IonTitle>
      </IonToolbar>
    </IonHeader>
    <IonContent class="ion-padding">
      <IonNote v-if="core" data-testid="reached">
        core on 127.0.0.1:{{ core.port }} — {{ core.dir }}
      </IonNote>
      <IonNote v-else-if="!trouble">starting the core…</IonNote>
      <p v-if="trouble" data-testid="trouble" style="color: #c00">{{ trouble }}</p>

      <IonItem>
        <IonInput v-model="title" placeholder="a title" data-testid="title" />
        <IonButton slot="end" data-testid="make" @click="make">new note</IonButton>
      </IonItem>

      <IonList data-testid="entries">
        <IonItem v-for="entry in entries" :key="entry.path">
          <IonLabel>
            {{ entry.name }}
            <IonNote v-if="entry.folder"> · folder</IonNote>
          </IonLabel>
        </IonItem>
      </IonList>
      <IonNote data-testid="count">{{ entries.length }} entries</IonNote>
    </IonContent>
  </IonApp>
</template>
