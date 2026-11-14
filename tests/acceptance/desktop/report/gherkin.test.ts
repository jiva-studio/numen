import { describe, expect, it } from 'vitest'
import { findScenario, parseFeature } from './gherkin.js'

const SOURCE = `Feature: Water plants
  Plants get water on their days.

  Background:
    Given the garden "Backyard"

  Scenario: A fern is watered
    When the gardener waters the plants:
      | plant | litres |
      | Fern  | 2      |
    Then the log says:
      """markdown
      Fern: 2 l
      """

  Rule: Cacti are dry

    Scenario Outline: A cactus waits
      When <days> days pass
      Then the cactus "<name>" is dry

      Examples:
        | days | name  |
        | 3    | Prick |
        | 9    | Spike |
`

describe('findScenario', () => {
  const feature = parseFeature(SOURCE, 'features/garden/water-plants.feature')

  it('gives the steps as written, the background first, with their table and doc string', () => {
    expect(findScenario(feature, 7)).toEqual({
      featureName: 'Water plants',
      name: 'A fern is watered',
      line: 7,
      steps: [
        { keyword: 'Given', text: 'the garden "Backyard"', line: 5 },
        {
          keyword: 'When',
          text: 'the gardener waters the plants:',
          line: 8,
          dataTable: [
            ['plant', 'litres'],
            ['Fern', '2'],
          ],
        },
        { keyword: 'Then', text: 'the log says:', line: 11, docString: { content: 'Fern: 2 l', mediaType: 'markdown' } },
      ],
    })
  })

  it('finds an outline by its example row inside a rule, with the values put in', () => {
    const scenario = findScenario(feature, 25)
    expect(scenario?.steps.map((step) => `${step.keyword} ${step.text}`)).toEqual([
      'Given the garden "Backyard"',
      'When 9 days pass',
      'Then the cactus "Spike" is dry',
    ])
  })

  it('finds nothing at a line where no scenario starts', () => {
    expect(findScenario(feature, 3)).toBeUndefined()
  })
})
