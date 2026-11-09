/** The sample files a scenario can put in a vault: a file name, and where the repository keeps that file. */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const E2E_FIXTURES = path.resolve(here, '../../../e2e/desktop/fixtures')
const VAULT_ASSETS = path.resolve(here, '../../../vault/assets')

const PDF_SAMPLES = new Map([
  ['flywheel.pdf', E2E_FIXTURES],
  ['paper.pdf', VAULT_ASSETS],
  ['pumped-storage.pdf', E2E_FIXTURES],
  ['regenerative-braking.pdf', E2E_FIXTURES],
  ['sample.pdf', E2E_FIXTURES],
  ['shore.pdf', E2E_FIXTURES],
])

/** The file of the repository that is the sample of a PDF, found by the PDF's file name. */
export const getPdfSample = (fileName: string): string => {
  const folder = PDF_SAMPLES.get(fileName)
  if (folder === undefined) {
    throw new Error(`there is no sample PDF "${fileName}": the samples are ${[...PDF_SAMPLES.keys()].join(', ')}`)
  }
  const sample = path.join(folder, fileName)
  if (!fs.existsSync(sample)) throw new Error(`the sample PDF "${fileName}" is missing at ${sample}`)
  return sample
}
