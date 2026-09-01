// Regression checks for src/game/legality.ts (no test runner in package.json).
// Run: node scripts/check-legality.mjs

function cardKey(c) {
  return c.suit + c.rank
}

function requiredLeadSuit(hand, trick) {
  if (!trick || trick.length === 0 || !hand || hand.length === 0) return null
  const first = trick[0]
  if (!first) return null
  const lead = first.card.suit
  const hasLead = hand.some((c) => c.suit === lead)
  return hasLead ? lead : null
}

function isMyTurn(opts) {
  return (
    opts.phase === 'trick_play' &&
    opts.turn === opts.you &&
    opts.trickLen < 4 &&
    !opts.matchOver
  )
}

function isCardLegal(card, opts) {
  if (!opts.myTurn) return false
  return opts.leadSuit === null || card.suit === opts.leadSuit
}

function assert(cond, msg) {
  if (!cond) {
    console.error('FAIL:', msg)
    process.exitCode = 1
  } else {
    console.log('ok:', msg)
  }
}

const hand = [
  { suit: 'spades', rank: 14 },
  { suit: 'hearts', rank: 10 },
  { suit: 'spades', rank: 2 },
]

assert(isMyTurn({ phase: 'trick_play', turn: 1, you: 1, trickLen: 0, matchOver: false }), 'my turn when leading')
assert(!isMyTurn({ phase: 'trick_play', turn: 2, you: 1, trickLen: 0, matchOver: false }), 'not my turn')
assert(!isMyTurn({ phase: 'trick_play', turn: 1, you: 1, trickLen: 4, matchOver: false }), 'blocked when trick full')

assert(requiredLeadSuit(hand, []) === null, 'no lead when empty trick')
assert(requiredLeadSuit(hand, [{ card: { suit: 'spades', rank: 5 } }]) === 'spades', 'must follow spades')
assert(requiredLeadSuit(hand, [{ card: { suit: 'clubs', rank: 5 } }]) === null, 'void in clubs -> any')

const lead = requiredLeadSuit(hand, [{ card: { suit: 'spades', rank: 5 } }])
assert(isCardLegal(hand[0], { myTurn: true, leadSuit: lead }), 'AS legal')
assert(!isCardLegal(hand[1], { myTurn: true, leadSuit: lead }), 'TH illegal when must follow')
assert(isCardLegal(hand[1], { myTurn: true, leadSuit: null }), 'TH legal when leading')
assert(!isCardLegal(hand[0], { myTurn: false, leadSuit: null }), 'illegal off turn')

assert(cardKey(hand[0]) === 'spades14', 'card key')

if (process.exitCode) {
  console.error('legality checks failed')
  process.exit(1)
}
console.log('all legality checks passed')
