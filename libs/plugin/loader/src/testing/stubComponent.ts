import { defineComponent } from 'vue'
import type { ConnectionHost } from '@numen/plugin-sdk'

/** A component that draws nothing, for a registration whose component a test never mounts. */
export const STUB_COMPONENT = defineComponent({ name: 'StubComponent', render: () => null })

/** Connections that end every call at once with nothing, for a view that needs them to be mounted. */
export const STUB_CONNECTIONS: ConnectionHost = {
  call: async function* () {
    yield* []
    return { ok: true, value: new Uint8Array() }
  },
}
