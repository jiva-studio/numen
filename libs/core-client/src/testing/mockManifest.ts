import { create } from '@bufbuild/protobuf'
import { ManifestSchema } from '@numen/protocol'
import type { Manifest } from '../manifest'

/** A manifest for a test: the fields given, and the schema's default for the rest. */
export function createMockManifest(
  id: string,
  fields: {
    name?: string
    version?: string
    permissions?: readonly string[]
    frontend?: boolean
  } = {},
): Manifest {
  return create(ManifestSchema, {
    id,
    name: fields.name ?? id,
    version: fields.version ?? '1.0.0',
    permissions: [...(fields.permissions ?? [])],
    ...(fields.frontend ? { frontend: { entrypoint: 'dist/index.js', styles: '' } } : {}),
  })
}
