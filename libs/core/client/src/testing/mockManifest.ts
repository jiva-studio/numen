import { create } from '@bufbuild/protobuf'
import { ManifestSchema } from '@numen/protocol'
import type { Manifest } from '../manifest'

/** What a manifest declares that its plugin registers, each as the ids the plugin registers. */
interface Declarations {
  readers?: readonly string[]
  tabKinds?: readonly string[]
  commands?: readonly string[]
  handlers?: readonly string[]
  settingsViews?: readonly string[]
  capabilities?: readonly string[]
}

function createContributes(fields: Declarations & { viewCapabilities?: readonly string[] }) {
  const { readers = [], tabKinds = [], commands = [], handlers = [] } = fields
  return {
    ...(fields.viewCapabilities ? { viewCapabilities: [...fields.viewCapabilities] } : {}),
    readers: readers.map((id) => ({ id })),
    tabKinds: tabKinds.map((kind) => ({ kind })),
    commands: commands.map((id) => ({ id })),
    handlers: handlers.map((command) => ({ command })),
  }
}

/** A manifest for a test: the fields given, and the schema's default for the rest. */
export function createMockManifest(
  id: string,
  fields: Declarations & {
    name?: string
    version?: string
    permissions?: readonly string[]
    frontend?: boolean
    viewCapabilities?: readonly string[]
  } = {},
): Manifest {
  const { name = id, version = '1.0.0', permissions = [], capabilities = [] } = fields
  const { settingsViews = [] } = fields
  return create(ManifestSchema, {
    id,
    name,
    version,
    permissions: [...permissions],
    capabilities: [...capabilities],
    settingsViews: settingsViews.map((view) => ({ id: view })),
    ...(fields.frontend ? { frontend: {} } : {}),
    contributes: createContributes(fields),
  })
}
