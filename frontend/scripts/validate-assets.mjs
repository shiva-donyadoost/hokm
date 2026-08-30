// Build-time asset validation (impliment.md §36): all 52 card faces + the
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
  console.error(`card asset validation FAILED — missing ${missing.length} assets:`)
  for (const f of missing) console.error(`  ${f}`)
  process.exit(1)
}
console.log(`card assets OK (${present.size} files validated)`)
