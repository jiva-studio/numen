import { describe, expect, it } from 'vitest'
import { countFilesByFolder, findMixedSamples } from './folder-samples.js'

describe('findMixedSamples', () => {
  it('finds no sample while the folder is whole in one place and then in the other', () => {
    const samples = [{ counts: { Food: 20, Meals: 0 } }, { counts: { Food: 0, Meals: 0 } }, { counts: { Food: 0, Meals: 20 } }]

    expect(findMixedSamples(samples, 'Food', 'Meals')).toStrictEqual([])
  })

  it('finds the sample that holds files of both folders', () => {
    const mixed = { counts: { Food: 12, Meals: 8 } }

    expect(findMixedSamples([{ counts: { Food: 20, Meals: 0 } }, mixed, { counts: { Food: 0, Meals: 20 } }], 'Food', 'Meals')).toStrictEqual([mixed])
  })

  it('takes a folder a sample does not name as empty', () => {
    expect(findMixedSamples([{ counts: { Food: 3 } }], 'Food', 'Meals')).toStrictEqual([])
  })
})

describe('countFilesByFolder', () => {
  it('counts the files under each folder, files of deeper folders too', () => {
    const paths = ['Food/File 1.bin', 'Food/Sub/File 2.bin', 'Meals/File 1.bin', 'Other/File 1.bin']

    expect(countFilesByFolder(paths, ['Food', 'Meals'])).toStrictEqual({ Food: 2, Meals: 1 })
  })

  it('counts a folder that holds nothing as zero and does not take a folder with the same start for it', () => {
    expect(countFilesByFolder(['Foods/File 1.bin'], ['Food', 'Meals'])).toStrictEqual({ Food: 0, Meals: 0 })
  })
})
