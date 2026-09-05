function isTrumpCutWin(trick, trump) {
  if (!trump) return false
  if (trick.lead_suit === trump) return false
  const winnerPlay = trick.cards.find((c) => c.seat === trick.winner)
  if (!winnerPlay) return false
  return winnerPlay.card.suit === trump
}
function mapTrickCompleted(trick, trump) {
  if (isTrumpCutWin(trick, trump)) return { type: 'TRUMP_CUT', seat: trick.winner }
  return { type: 'TRICK_WON', seat: trick.winner }
}
function mapEngineEvent(ev) {
  const p = ev.payload || {}
  switch (ev.name) {
    case 'hakem_selected': return [{ type: 'HAKEM_SELECTED', seat: Number(p.seat || 0) }]
    case 'next_round_started': return [{ type: 'HAKEM_SELECTED', seat: Number(p.hakem || 0) }]
    case 'trump_selected': return [{ type: 'TRUMP_SELECTED', suit: String(p.suit || '') }]
    case 'card_played': return [{ type: 'CARD_PLAYED', seat: Number(p.seat || 0) }]
    default: return []
  }
}
function dealCountsForHandJump(prevLen, nextLen) {
  if (prevLen === 0 && nextLen === 5) return 5
  if (prevLen === 5 && nextLen === 13) return 8
  return 0
}
function assert(cond, msg) {
  if (!cond) { console.error('FAIL:', msg); process.exitCode = 1 }
  else console.log('ok:', msg)
}
const cutTrick = {
  number: 3, lead_suit: 'spades', winner: 2, winner_team: 0,
  cards: [
    { seat: 0, card: { suit: 'spades', rank: 10 } },
    { seat: 1, card: { suit: 'spades', rank: 5 } },
    { seat: 2, card: { suit: 'hearts', rank: 2 } },
    { seat: 3, card: { suit: 'clubs', rank: 14 } },
  ],
}
assert(isTrumpCutWin(cutTrick, 'hearts') === true, 'trump cut when trump wins off-lead')
assert(isTrumpCutWin(cutTrick, 'spades') === false, 'not a cut when trump is lead')
assert(isTrumpCutWin(cutTrick, null) === false, 'no cut without trump')
assert(mapTrickCompleted(cutTrick, 'hearts').type === 'TRUMP_CUT', 'map cut')
assert(mapTrickCompleted(cutTrick, 'spades').type === 'TRICK_WON', 'map non-cut')
assert(mapEngineEvent({ name: 'hakem_selected', payload: { seat: 1 } })[0].type === 'HAKEM_SELECTED', 'hakem')
assert(mapEngineEvent({ name: 'next_round_started', payload: { hakem: 3 } })[0].seat === 3, 'next round hakem')
assert(mapEngineEvent({ name: 'trump_selected', payload: { suit: 'clubs' } })[0].suit === 'clubs', 'trump')
assert(mapEngineEvent({ name: 'card_played', payload: { seat: 2 } })[0].type === 'CARD_PLAYED', 'card played')
assert(mapEngineEvent({ name: 'round_completed', payload: {} }).length === 0, 'ignore round_completed')
assert(dealCountsForHandJump(0, 5) === 5, 'initial deal')
assert(dealCountsForHandJump(5, 13) === 8, 'remaining deal')
assert(dealCountsForHandJump(5, 6) === 0, 'ignore non-deal')
if (process.exitCode) { console.error('audio mapping checks failed'); process.exit(1) }
console.log('all audio mapping checks passed')