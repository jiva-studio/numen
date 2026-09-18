/**
 * The controls a recording is played by: what they say, and what they say a
 * person did.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Player from './Player.vue'

type PlayerProps = InstanceType<typeof Player>['$props']

const mountPlayer = (props: Partial<PlayerProps> = {}) =>
  mount(Player, { props: { at: 0, length: 125_000, isPlaying: false, ...props } })

const button = (player: ReturnType<typeof mountPlayer>) => player.get('button')
const bar = (player: ReturnType<typeof mountPlayer>) => player.get('input[type="range"]')
const times = (player: ReturnType<typeof mountPlayer>) =>
  player.findAll('.player__at').map((one) => one.text())

describe('what a person is told', () => {
  it('offers to start it while nothing is playing, and to stop it while it is', () => {
    const still = mountPlayer()
    expect(button(still).attributes('aria-label')).toBe('Play')
    expect(button(still).attributes('aria-pressed')).toBe('false')

    const going = mountPlayer({ isPlaying: true })
    expect(button(going).attributes('aria-label')).toBe('Pause')
    expect(button(going).attributes('aria-pressed')).toBe('true')
  })

  it('says where the sound stands and how long it runs, as a person reads a clock', () => {
    expect(times(mountPlayer({ at: 65_400 }))).toEqual(['1:05', '2:05'])
  })

  it('is named in the words it is given', () => {
    const player = mountPlayer({ label: 'Утренняя запись', play: 'Играть' })
    expect(player.get('[role="group"]').attributes('aria-label')).toBe('Утренняя запись')
    expect(bar(player).attributes('aria-label')).toBe('Утренняя запись')
    expect(button(player).attributes('aria-label')).toBe('Играть')
  })

  // The bar is a row of numbers to a reader who is listening, so where it
  // stands is read out as the clock says it.
  it('reads the bar out as a time and not as a count of milliseconds', () => {
    expect(bar(mountPlayer({ at: 65_400 })).attributes('aria-valuetext')).toBe('1:05')
  })
})

describe('what a person did', () => {
  it('asks for it to start, and to stop once it is playing', async () => {
    const still = mountPlayer()
    await button(still).trigger('click')
    expect(still.emitted('play')).toHaveLength(1)
    expect(still.emitted('pause')).toBeUndefined()

    const going = mountPlayer({ isPlaying: true })
    await button(going).trigger('click')
    expect(going.emitted('pause')).toHaveLength(1)
    expect(going.emitted('play')).toBeUndefined()
  })

  it('asks for the sound to be put where the bar was moved to, in milliseconds', async () => {
    const player = mountPlayer()
    await bar(player).setValue('42000')
    expect(player.emitted('seek')).toEqual([[42_000]])
  })
})

describe('a sound longer than it was said to be', () => {
  // What plays says how far in it is, and it can be further in than whatever
  // said how long it was. The bar runs that far rather than stopping short.
  it('runs as far as the getFurthest anything has stood in it', () => {
    const player = mountPlayer({ at: 200_000, length: 125_000 })
    expect(bar(player).attributes('max')).toBe('200000')
    expect((bar(player).element as HTMLInputElement).value).toBe('200000')
    expect(times(player)[1]).toBe('3:20')
  })

  it('runs nowhere where nothing said how long it is', () => {
    const player = mountPlayer({ at: 0, length: 0 })
    expect(bar(player).attributes('max')).toBe('0')
    expect(times(player)).toEqual(['0:00', '0:00'])
  })
})
