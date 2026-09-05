// Build-time asset validation (impliment.md): all 52 card faces + the
// card back must exist. Missing assets fail the build with a clear error
// instead of rendering broken images at runtime.
import { readdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import process from 'node:process'

const RANKS = ['2', '3', '4', '5', '6', '7', '8', '9', '10', 'jack', 'queen', 'king', 'ace']
const SUITS = ['spades', 'hearts', 'diamonds', 'clubs']

const dir = join(dirname(fileURLToPath(import.meta.url)), '..', 'public', 'cards')
const present = new Set(readdirSync(dir))

const missing = []
for (const s of SUITS) {
  for (const r of RANKS) {
    const f = `${r}_of_${s}.svg`
    if (!present.has(f)) missing.push(f)
  }
}
if (!present.has('card-back.svg')) missing.push('card-back.svg')

if (missing.length > 0) {
  console.error(`card asset validation FAILED - missing ${missing.length} assets:`)
  for (const f of missing) console.error(`  ${f}`)
  process.exit(1)
}
console.log(`card assets OK (${present.size} files validated)`)

// Audio SFX (ADR-0019 / ADR-0020) — source of truth is src/assets/audio
// (Vite-imported + inlined). public/assets/audio is kept as a replaceable
// mirror with the same filenames.
const root = dirname(fileURLToPath(import.meta.url))
const audioDirs = [
  join(root, '..', 'src', 'assets', 'audio'),
  join(root, '..', 'public', 'assets', 'audio'),
]
const audioRequired = [
  'card-deal-01.wav', 'card-deal-01.ogg',
  'card-deal-02.wav', 'card-deal-02.ogg',
  'card-deal-03.wav', 'card-deal-03.ogg',
  'card-play-01.wav', 'card-play-01.ogg',
  'card-play-02.wav', 'card-play-02.ogg',
  'card-play-03.wav', 'card-play-03.ogg',
  'hakem-selected.wav', 'hakem-selected.ogg',
  'trump-selected.wav', 'trump-selected.ogg',
  'trump-cut.wav', 'trump-cut.ogg',
  'trick-won.wav', 'trick-won.ogg',
  'card-collect.wav', 'card-collect.ogg',
]
for (const audioDir of audioDirs) {
  const audioPresent = new Set(readdirSync(audioDir))
  const audioMissing = audioRequired.filter((f) => !audioPresent.has(f))
  if (audioMissing.length > 0) {
    console.error('audio asset validation FAILED in ' + audioDir + ' - missing ' + audioMissing.length + ' assets:')
    for (const f of audioMissing) console.error('  ' + f)
    process.exit(1)
  }
  console.log('audio assets OK in ' + audioDir + ' (' + audioRequired.length + ' files validated)')
}
