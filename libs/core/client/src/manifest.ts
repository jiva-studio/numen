/**
 * The manifest of a plugin as the application reads it: the generated protocol types and the reader of
 * manifest.json.
 */
import { fromJson, type JsonValue } from '@bufbuild/protobuf'
import { ManifestSchema, type Manifest as ManifestMessage } from '@numen/protocol'
import { asFailure, asValue, formatErrorMessage, type Result } from '@numen/result'

/** The schema of a plugin's manifest.json. */
export type Manifest = ManifestMessage

/**
 * The text of a manifest.json read against the generated schema. Keys the schema does not name are
 * ignored; a value of the wrong type is an error.
 */
export const parseManifest = (text: string): Result<Manifest, string> => {
  try {
    const document = JSON.parse(text) as JsonValue
    return asValue(fromJson(ManifestSchema, document, { ignoreUnknownFields: true }))
  } catch (error) {
    return asFailure(formatErrorMessage(error))
  }
}
