SVG path for front card: C:\Users\donyadoost\Downloads\Playing Cards\SVG-cards-1.3

its' more then 52 use right


# IMPLEMENTATION PROMPT — HOKM GAMEPLAY UX, TIMING, PERSISTENCE & CHAT ENHANCEMENTS

You are the senior full-stack engineer responsible for implementing the following enhancements to the existing Iranian Hokm multiplayer game.

You MUST follow every rule defined in `AGENTS.md`.

Do not treat this as a quick UI modification. These changes affect the game engine, server-authoritative state management, WebSocket synchronization, frontend UX, persistence, room configuration, animation timing, chat, and E2E testing.

---

# 1. NON-NEGOTIABLE CONFIGURATION RULE

## NO HARD-CODED GAMEPLAY VALUES

This is a global hard rule.

Every number, duration, timeout, animation duration, gameplay limit, and configurable game parameter MUST be represented as a configuration value or constant with a clear domain meaning.

Do NOT scatter numeric literals throughout the code.

Examples of values that MUST NOT be hard-coded inside gameplay logic:

* Card selection timeout
* Hakem selection timeout
* Fast/Medium/Slow timeout values
* Animation duration
* Trick winner display duration
* Reconnection grace period
* AI takeover timeout
* Number of rounds
* Number of tricks
* Number of players
* Number of cards
* Number of initial cards
* Number of cards per player
* Chat limits
* Rate limits
* Any future gameplay timing value

Use named configuration values.

For example, conceptually:

```text
GameConfig
    PlayerCount
    CardsPerPlayer
    InitialCards
    TricksPerRound
    RoundCount
    HakemSelectionTimeout
    CardSelectionTimeout
    TrickWinnerDisplayDuration
    CardPlayAnimationDuration
    ReconnectGracePeriod
```

Do not use literal values such as:

```text
10
15
30
3
5
13
52
0.5
```

directly inside business logic.

Default values are allowed, but they must exist in one clearly defined configuration layer.

The architecture MUST allow these values to be changed later without modifying gameplay logic.

---

# 2. FIRST ACTION

Before modifying code:

1. Read `AGENTS.md`.
2. Inspect the existing repository.
3. Inspect the current Game Engine.
4. Inspect current Game State.
5. Inspect current Room implementation.
6. Inspect WebSocket architecture.
7. Inspect frontend architecture.
8. Inspect authentication/session persistence.
9. Inspect current chat implementation.
10. Inspect current card assets.
11. Inspect existing tests.
12. Inspect existing E2E tests.
13. Inspect existing Docker configuration.
14. Review previous ADRs.
15. Review previous Lessons Learned.

Do not assume the existing architecture.

Understand it first.

---

# 3. CREATE A CHANGE PLAN

Before implementation, create or update:

`docs/IMPLEMENTATION_PLAN.md`

Add a dedicated section for these enhancements.

Break the work into logical phases.

At minimum:

## Phase A — Configuration Architecture

Create centralized configuration for:

* Gameplay values
* Timeouts
* Animation timing
* Room settings
* Reconnection
* AI takeover
* Round count

## Phase B — Game Timing and State Machine

Implement authoritative timers and timeout transitions.

## Phase C — Card Dealing and Animation

Implement animated dealing and private card visibility.

## Phase D — Sequential Trick Presentation

Slow down visual presentation of each played card.

## Phase E — Trick Winner Animation

Implement winner highlighting and card collection animation.

## Phase F — Round/Hand Score Display

Display previous round winners.

## Phase G — Player Card Interaction

Implement tap-to-select, tap-to-confirm, drag-and-drop, and drag cancellation.

## Phase H — Persistent Game Session

Implement refresh persistence and WebSocket reconnection.

## Phase I — AI Takeover

Implement configurable disconnect timeout and AI takeover.

## Phase J — Chat Improvements

Implement emoji picker, unread badge, and room-level chat toggle.

## Phase K — SVG Card Assets

Integrate the provided card assets and card-back asset.

## Phase L — Testing

Implement Unit, Integration, and E2E tests for all changes.

Each phase must be tested and committed before proceeding.

---

# 4. ROUND TERMINOLOGY

Use precise domain terminology.

In this project:

A "Round" means one complete game cycle in which all players receive their full set of cards and all required tricks are played.

For standard Hokm:

```text
1 Round
    ↓
13 Tricks
```

The Room creator can configure the match to contain:

```text
1 Round
3 Rounds
5 Rounds
```

Do not confuse:

* Trick
* Round
* Match

Use these concepts consistently throughout the codebase.

Prefer:

```text
Trick
Round
Match
```

over ambiguous names such as:

```text
Hand
Game
Turn
```

unless the existing architecture has a justified domain-specific naming convention.

---

# 5. MATCH ROUND COUNT

The Room creator must be able to choose the number of Rounds before the match starts.

Supported default options:

* 1 Round
* 3 Rounds
* 5 Rounds

However, these values MUST NOT be hard-coded into game logic.

Create a configurable setting such as:

```text
RoundCount
```

The architecture should allow future values without rewriting the Game Engine.

Example future configuration:

```text
RoundCount = 7
```

should be possible without changing the Game Engine.

---

# 6. CARD DEALING ANIMATION

At the beginning of a Round:

Cards must be dealt with an animation.

The animation should visually communicate cards being distributed to players.

The dealing must remain synchronized with the authoritative Game State.

Important:

The animation is presentation logic.

The actual card distribution is determined by the server.

Do not let animation timing determine game legality.

---

# 7. PRIVATE CARD VISIBILITY

Each player must only see their own cards.

Opponent cards must remain hidden.

Use the card-back asset for hidden cards.

The server must never send hidden opponent card information to the client.

The frontend should receive something conceptually similar to:

```text
My Hand:
    actual cards

Opponent:
    card count + card backs

Partner:
    card count + card backs
```

Do not expose the actual opponent card identities through WebSocket payloads.

---

# 8. HAKEM TRUMP SELECTION TIMER

After the initial cards have been dealt:

The Hakem must choose Trump.

The selection timer must start when the Hakem becomes eligible to choose Trump.

The default selection duration is configurable.

Do NOT hard-code the timeout.

The current default product requirement is:

```text
10 seconds
```

but this must come from configuration.

If the Hakem does not choose Trump before the timeout:

1. Server determines a legal Trump automatically.
2. The selection must be deterministic or based on a configurable strategy.
3. Do not make the automatic decision dependent on frontend behavior.
4. Broadcast the selected Trump to all players.
5. Record that the selection was automatic.

The automatic selection behavior must be configurable.

---

# 9. HAKEM TIMER UI

Under the Hakem's name, display a progress bar.

The progress bar must represent the remaining Trump-selection time.

Requirements:

* Visible to all relevant players.
* Clearly indicate whose turn it is.
* Smoothly animate.
* Reach zero at the authoritative server deadline.
* Not rely solely on client-side countdown logic.

The server timestamp/deadline should be authoritative.

Frontend should calculate visual remaining time using synchronized server time where appropriate.

---

# 10. CARD SELECTION TIMER

Every player must have a configurable amount of time to play a card.

The Room creator selects the game speed:

### Fast

Default card-selection timeout:

```text
5 seconds
```

### Medium

Default:

```text
10 seconds
```

### Slow

Default:

```text
15 seconds
```

These are defaults only.

They MUST be stored in configuration.

Never implement:

```text
if speed == "fast" => 5
```

through scattered magic numbers.

Prefer a configuration model:

```text
GameSpeedConfig
    Fast
    Medium
    Slow
```

where each contains a configurable timeout.

---

# 11. CARD TIMEOUT BEHAVIOR

If the player does not play a card before the authoritative timeout:

The server automatically selects a card.

IMPORTANT:

The card must NOT be random.

The server must:

1. Determine all legal cards.
2. Sort/evaluate legal cards according to the configured automatic-play strategy.
3. Select the lowest legal card.
4. Play it on behalf of the player.
5. Broadcast the action.
6. Mark the action as automatic.

Conceptually:

```text
Legal Cards
    ↓
Find lowest legal card
    ↓
Play automatically
```

The automatic-play strategy should be isolated behind an interface so that it can later be changed.

For example:

```text
TimeoutCardStrategy
```

Possible future strategies:

* LowestLegalCard
* Defensive
* AI
* ConfigurableStrategy

The default must be:

```text
LowestLegalCard
```

---

# 12. AUTHORITATIVE TIMER

Timers must be server-authoritative.

Do not trust the browser clock.

The server should establish:

```text
deadline
```

and send the relevant timestamp/state to clients.

The client renders the countdown.

If the client is delayed or disconnected, the server remains authoritative.

When the deadline expires:

```text
Server
    ↓
Validate timeout
    ↓
Perform automatic action
    ↓
Broadcast event
```

---

# 13. SEQUENTIAL CARD PLAY PRESENTATION

The current game presentation is too fast.

When each player plays a card:

The card must remain visually understandable before the next player acts.

However:

The actual animation duration for card movement should be a separate configurable presentation setting.

Current default requirement:

```text
0.5 seconds
```

Do not hard-code this value.

Example:

```text
CardPlayAnimationDuration
```

must be configurable.

Do NOT couple gameplay timeout values with animation duration.

---

# 14. TRICK WINNER PRESENTATION

After all four players have played their cards:

The winner must be clearly identified.

Do not immediately remove the cards.

Current default winner-display duration:

```text
3 seconds
```

This must be configurable.

During this state:

* Highlight winner.
* Show appropriate visual feedback.
* Clearly indicate which player won the Trick.

---

# 15. CARD COLLECTION ANIMATION

After the winner is determined:

The four cards on the table must animate toward the winning player.

Desired visual sequence:

```text
Four cards on table
        ↓
Winner highlighted
        ↓
Cards visually stack/collect
        ↓
Cards move toward winner
        ↓
Trick completes
```

The animation must:

* Preserve visual clarity.
* Stack the cards visually.
* Move the stack toward the winner.
* Complete before the next Trick starts, unless the configuration explicitly allows otherwise.

The actual winner is determined by the server.

Animation must never determine the winner.

---

# 16. TRICK STATE MACHINE

Introduce explicit presentation/game states where necessary.

Conceptually:

```text
PLAYING_TRICK
        ↓
ALL_CARDS_PLAYED
        ↓
WINNER_REVEAL
        ↓
COLLECTING_CARDS
        ↓
NEXT_TRICK
```

Do not use arbitrary frontend timers to guess when the next Trick begins.

The frontend should respond to server events/state.

---

# 17. CURRENT TRUMP DISPLAY

The currently selected Trump must always be clearly visible at the top of the game screen.

Display:

* Trump Suit
* Appropriate visual icon
* Accessible text label

The Trump indicator must update immediately after selection.

---

# 18. ROUND WINNER DISPLAY

During a multi-Round Match, the UI must show the previous Round results.

Example:

If the Match is configured for 3 Rounds and Round 2 is currently being played:

```text
Round 1
🏆 Team A

Round 2
Currently Playing

Round 3
Not Started
```

At minimum show:

* Current Round
* Total Rounds
* Previous Round Winner
* Current Match Score

The display must remain understandable on mobile.

---

# 19. MATCH SCORE

The top area of the game UI should clearly show:

```text
Team A
Rounds Won: X

Team B
Rounds Won: Y

Round X / Total
```

All values must come from the authoritative server state.

Do not calculate authoritative scores exclusively in the frontend.

---

# 20. CARD HAND UI — ARC LAYOUT

The current player's cards must be displayed in an arc/fan layout.

Requirements:

* Responsive
* Mobile-first
* Touch-friendly
* Cards should overlap appropriately
* The selected card should visually rise/highlight
* The layout must adapt to screen width
* Cards must remain individually selectable

Avoid making the hand unusable on small screens.

---

# 21. TAP-TO-SELECT / TAP-TO-PLAY

Implement two-step tap interaction.

First tap:

```text
Card becomes selected/highlighted
```

Second tap on the selected card:

```text
Card is submitted for play
```

If the card is illegal:

* Do not send it as a valid gameplay action.
* Show appropriate feedback.
* Server validation remains authoritative.

If another card is selected:

The previous selection should return to its normal state.

---

# 22. DRAG AND DROP

Support drag-and-drop card play.

Desktop:

* Mouse drag
* Drag toward the table
* Release to play

Mobile:

* Touch/pointer drag
* Card follows the pointer/finger
* Drag toward the play area
* Release to play

Use pointer events or an appropriate abstraction so the same interaction model works across devices.

---

# 23. DRAG CANCELLATION

The player must be able to cancel a drag.

If the user starts dragging a card but changes their mind:

```text
Drag
 ↓
Move away / cancel
 ↓
Return card
 ↓
Snap card back to original arc position
```

The card must animate smoothly back into its original position.

No gameplay action should be sent when the drag is cancelled.

---

# 24. ILLEGAL CARD UX

The frontend may identify obviously illegal cards for UX purposes.

Examples:

* Dim illegal cards
* Prevent interaction
* Show a short message

However:

The server remains authoritative.

Never rely on frontend legality checks.

If a malicious or outdated client submits an illegal card:

```text
Server rejects action
```

and sends an appropriate error.

---

# 25. REFRESH PERSISTENCE

Refreshing the browser must NOT log the user out.

After refresh:

1. Authentication state is restored.
2. User remains logged in.
3. Active room session is restored if applicable.
4. Active game session is restored if applicable.
5. WebSocket reconnects.
6. Server sends the correct Player View.
7. User returns to the correct screen/state.

The player should not lose their active game simply because of a browser refresh.

---

# 26. AUTHENTICATION PERSISTENCE

Use the existing authentication architecture if sound.

If authentication currently depends on volatile browser state, improve it.

Use secure persistent session/token handling appropriate for the existing architecture.

Do not store sensitive secrets insecurely.

Refresh must not cause:

```text
logout
```

unless the authentication session has genuinely expired or been revoked.

---

# 27. GAME RECONNECTION

A WebSocket connection must NOT be the same thing as a player session.

When a player reconnects:

```text
Authentication
    ↓
Identify Player
    ↓
Identify active Game
    ↓
Reattach connection
    ↓
Send Player-specific Game State
    ↓
Continue
```

The current Game State must be restored from the authoritative source.

---

# 28. DISCONNECT GRACE PERIOD

When a player disconnects:

The game should NOT immediately replace them with AI.

Use a configurable reconnect grace period.

Current default:

```text
30 seconds
```

This must NOT be hard-coded.

Conceptually:

```text
Player disconnected
        ↓
Grace period starts
        ↓
Player reconnects?
       / \
     YES  NO
      ↓    ↓
 Continue  AI takeover
```

---

# 29. AI TAKEOVER

If the player does not reconnect before the configurable grace period expires:

AI takes over that player's seat.

The game continues.

The AI must use exactly the same information boundary as the original player.

The AI must not gain access to hidden cards beyond what the player legitimately knew.

AI takeover must be represented as an explicit server-side state transition.

---

# 30. AI RETURN / RECONNECT

If a player reconnects after AI takeover:

Define and implement a safe policy.

The preferred behavior is:

* Reconnect to the same Player identity.
* Restore visibility of their own cards.
* Do not reveal hidden information.
* Do not corrupt the current game state.
* AI should stop controlling the seat only at a safe state boundary.

If the existing game architecture requires a different behavior, document the decision in an ADR.

---

# 31. CHAT ENABLE/DISABLE

The Room creator must be able to configure:

```text
Chat Enabled
Chat Disabled
```

This is a Room setting.

If Chat is disabled:

* Chat UI must NOT be visible to players.
* Chat button must not appear.
* Chat messages must not be accepted.
* Chat-related WebSocket events should not be unnecessarily sent.
* Backend must enforce the setting.

Do not implement this as a frontend-only visibility toggle.

---

# 32. CHAT EMOJI PICKER

Add an emoji picker using a React-compatible emoji library.

The implementation should use a maintained and appropriate library.

Do not implement an emoji picker manually unless there is a strong architectural reason.

The emoji picker should:

* Work on mobile
* Work on desktop
* Be accessible
* Not overflow the viewport
* Close appropriately
* Insert the selected emoji into the message input

Keep the emoji library isolated behind the Chat UI layer.

---

# 33. CHAT UNREAD NOTIFICATION BADGE

Add an unread notification badge to the Chat button.

Example:

```text
💬 3
```

Behavior:

* Increment when a new message arrives while Chat is closed.
* Reset when the user opens/reads the Chat.
* Do not increment for the user's own messages.
* Preserve unread state during normal UI state changes.
* Handle WebSocket reconnect appropriately.

If Chat is disabled:

There should be no Chat button and no Chat badge.

---

# 34. CARD SVG ASSETS

The project already has a folder containing SVG card images.

Inspect the provided assets before modifying the card rendering system.

Use those SVGs as the actual card artwork instead of generating card faces using CSS or another image set.

Create a clean asset mapping:

```text
Card
    ↓
Card Asset Resolver
    ↓
SVG asset
```

Do not scatter file paths throughout components.

Create a centralized mapping/resolver.

---

# 35. CARD BACK ASSET

Create a dedicated configurable location for the card-back asset.

For example:

```text
assets/cards/card-back.svg
```

or:

```text
assets/cards/card-back.png
```

Do not assume a specific format before inspecting the repository.

The card-back should be referenced through the same asset abstraction.

---

# 36. ASSET VALIDATION

At development/build time, validate that required card assets exist.

At minimum:

* All 52 card faces
* Card back

Missing assets should produce a clear error.

Do not silently render broken images.

---

# 37. RESPONSIVE CARD RENDERING

The card system must support:

* Mobile
* Tablet
* Desktop

The same card asset should be reusable across:

* Player hand
* Trick area
* Opponent hidden cards
* Animations
* Winner collection

Avoid duplicating card rendering implementations.

---

# 38. ANIMATION ARCHITECTURE

Create a reusable animation system.

Do not scatter arbitrary CSS transition durations throughout the application.

Centralize animation configuration.

Example conceptual configuration:

```text
AnimationConfig
    CardDealDuration
    CardPlayDuration
    TrickWinnerDisplayDuration
    CardCollectionDuration
    CardReturnDuration
```

All defaults must be configurable.

Current product defaults include:

```text
CardPlayDuration = 0.5 seconds
TrickWinnerDisplayDuration = 3 seconds
```

but these values must live in configuration.

---

# 39. GAMEPLAY VS PRESENTATION TIMING

Keep these separate.

Gameplay timing:

* Hakem timeout
* Card selection timeout
* Reconnection timeout

Presentation timing:

* Card deal animation
* Card play animation
* Winner reveal
* Card collection
* Drag cancellation animation

Changing an animation duration must NOT accidentally change the gameplay timeout.

---

# 40. WEB SOCKET EVENTS

Extend the WebSocket event system where necessary.

Possible events:

```text
ROUND_STARTED
CARDS_DEALING_STARTED
INITIAL_CARDS_DEALT
HAKEM_SELECTION_STARTED
HAKEM_SELECTION_TICK
TRUMP_SELECTED
TRUMP_SELECTION_TIMEOUT
CARDS_DEALT
TURN_STARTED
CARD_PLAYED
CARD_PLAY_TIMEOUT
TRICK_COMPLETED
TRICK_WINNER_REVEALED
CARDS_COLLECTING
ROUND_COMPLETED
MATCH_SCORE_UPDATED
PLAYER_DISCONNECTED
PLAYER_RECONNECTED
AI_TAKEOVER
CHAT_MESSAGE
CHAT_UNREAD
```

Do not create unnecessary high-frequency timer events if the client can calculate remaining time from a server-provided deadline.

Prefer:

```text
deadline timestamp
```

over broadcasting every second.

---

# 41. SERVER-SIDE GAME TIMER DESIGN

Use a reliable server-side timer mechanism.

Do not create one uncontrolled goroutine/timer per UI component.

Timers must belong to the Game/Room lifecycle.

Ensure timers are cancelled when:

* State changes
* Game ends
* Player acts
* Room closes
* Server shuts down

Avoid goroutine leaks.

---

# 42. TIMER RACE CONDITIONS

Handle these cases:

### Player plays exactly before timeout

Only one action should be accepted.

### Player action arrives after timeout

Server should reject/ignore the late manual action if automatic action has already been committed.

### Player disconnects during timer

Timer behavior must remain correct.

### Player reconnects during timer

Remaining time must be restored correctly.

### Server restart

If active game state is persisted, timers must be reconstructed correctly.

---

# 43. DATABASE / REDIS STATE

If the existing project uses Redis for active Game State:

Store enough information to reconstruct:

* Current phase
* Current round
* Current trick
* Current turn
* Timer deadline
* Trump
* Scores
* Player connection state
* AI takeover state

Do not persist ephemeral animation state as authoritative gameplay state.

---

# 44. E2E TESTING

Every user-visible feature in this change set MUST have E2E coverage where applicable.

Use the existing E2E framework if one exists.

If none exists, use an appropriate browser E2E framework such as Playwright.

At minimum implement:

## Card Dealing

Verify animated dealing starts and players see only their own cards.

## Hakem

Verify:

* Hakem receives initial cards.
* Hakem can select Trump.
* Timer is visible.
* Timeout causes automatic Trump selection.

## Card Selection

Verify:

* Timer starts on player's turn.
* Legal card can be played.
* Illegal card cannot be successfully submitted.
* Timeout causes the lowest legal card to be played automatically.

## Trick Winner

Verify:

* Four cards appear.
* Winner is identified.
* Winner remains visible for configured duration.
* Cards collect toward winner.
* Next Trick begins.

## Round Score

Verify previous Round winner is displayed during the next Round.

## Multi-Round Match

Verify:

* 1 Round configuration
* 3 Round configuration
* 5 Round configuration

and ensure the Match progresses correctly.

## Refresh

Verify:

```text
Login
→ Join Game
→ Refresh
→ Remain authenticated
→ Reconnect WebSocket
→ Restore game state
```

## Reconnection

Verify:

```text
Disconnect
→ reconnect within grace period
→ restore player
```

## AI Takeover

Verify:

```text
Disconnect
→ wait for configured grace period
→ AI takeover
→ game continues
```

## Chat

Verify:

* Chat enabled → Chat visible
* Emoji picker works
* New message creates unread badge
* Opening Chat clears unread badge
* Chat disabled → Chat completely hidden

## Card Interaction

Verify:

* Tap once selects
* Tap twice plays
* Drag plays
* Cancelled drag restores card position

---

# 45. UNIT TESTS

Add unit tests for every changed domain behavior.

At minimum:

### Configuration

* Default values load correctly.
* Configuration can be overridden.
* No gameplay logic depends on magic numbers.

### Timers

* Deadline creation
* Timeout detection
* Cancellation
* Race conditions

### Automatic Card Selection

* Only legal cards considered.
* Lowest legal card selected.
* No random selection.
* Follow-Suit rules respected.

### Hakem

* Hakem selection timer.
* Automatic Trump selection.
* Manual selection cancels timeout.

### Round

* Round count.
* Round completion.
* Previous winner persistence.

### Reconnection

* Session restoration.
* Player identity preservation.
* AI takeover.

### Assets

* 52 card assets resolve correctly.
* Card-back asset resolves.

---

# 46. INTEGRATION TESTS

Test integration between:

* Game Engine + Timer
* Game Engine + WebSocket
* Room + Game
* Redis + Game State
* Authentication + Session
* Chat + Room Settings
* AI + Game Engine

---

# 47. REGRESSION TESTS

Before considering the change complete:

Run the entire existing test suite.

The following existing behavior must continue working:

* Room creation
* Room joining
* Lobby
* Ready state
* Game start
* Hakem selection
* Trump
* Card dealing
* Legal moves
* Trick winner
* Scoring
* AI
* Chat
* Authentication

Do not regress existing functionality.

---

# 48. ACCESSIBILITY

Ensure:

* Timer has accessible semantics.
* Winner state is understandable without animation alone.
* Trump has text alternative.
* Buttons have labels.
* Cards have accessible labels.
* Chat button exposes unread count.
* Emoji picker is keyboard accessible where possible.

Animation should respect reduced-motion preferences.

If `prefers-reduced-motion` is enabled:

* Reduce non-essential animations.
* Preserve gameplay clarity.
* Never alter game rules or timing.

---

# 49. PERFORMANCE

Do not create unnecessary React re-renders for every timer tick.

Prefer calculating the displayed remaining time locally from:

```text
serverDeadline - synchronizedCurrentTime
```

Use efficient animation techniques.

Avoid layout thrashing during card movement.

Do not repeatedly manipulate DOM layout synchronously during animations.

---

# 50. MOBILE UX

Test on small mobile viewport sizes.

Ensure:

* Card arc fits the screen.
* Cards remain selectable.
* Drag interaction works.
* Chat does not overflow.
* Emoji picker stays inside viewport.
* Timer remains readable.
* Trump remains visible.
* Score remains visible.
* Winner animation does not hide critical information.

---

# 51. DOCUMENTATION

Update:

`docs/GAME_RULES.md`

`docs/ARCHITECTURE.md`

`docs/API.md`

`docs/WEBSOCKET.md`

`docs/AI.md`

where relevant.

Document all new configuration options.

Example:

```text
GAME_HAKEM_SELECTION_TIMEOUT
GAME_CARD_SELECTION_TIMEOUT_FAST
GAME_CARD_SELECTION_TIMEOUT_MEDIUM
GAME_CARD_SELECTION_TIMEOUT_SLOW
GAME_RECONNECT_GRACE_PERIOD
GAME_CARD_PLAY_ANIMATION_DURATION
GAME_TRICK_WINNER_DISPLAY_DURATION
GAME_ROUND_COUNT
GAME_CHAT_ENABLED
```

The actual configuration naming convention should match the existing project conventions.

---

# 52. ADR REQUIREMENTS

Create/update ADRs for significant architectural decisions.

At minimum evaluate whether ADRs are needed for:

* Centralized gameplay configuration
* Server-authoritative timers
* Reconnection/AI takeover
* Animation state architecture
* Card asset abstraction
* Chat configuration architecture
* Frontend drag/touch interaction architecture

Do not create meaningless ADRs.

---

# 53. LESSONS LEARNED

If any implementation error, failed test, race condition, incorrect assumption, configuration problem, Docker problem, WebSocket issue, UI issue, or architectural problem occurs:

1. Reproduce it.
2. Identify root cause.
3. Fix it.
4. Add a regression test.
5. Add a Lesson Learned to `agents.md`.
6. Commit the change.

Never silently fix and forget.

---

# 54. GIT COMMITS

Create a meaningful commit for every coherent change.

Examples:

```text
feat(game): add configurable gameplay timing

feat(game): add server authoritative card timeout

feat(game): select lowest legal card on timeout

feat(ui): add animated card dealing

feat(ui): add trick winner collection animation

feat(ui): add arc card hand interaction

feat(ui): add drag cancellation

feat(auth): restore session after refresh

feat(game): add configurable ai takeover

feat(chat): add emoji picker

feat(chat): add unread message badge

feat(chat): support room chat toggle

feat(assets): integrate svg card assets

test(e2e): cover card timeout flow

test(e2e): cover game reconnection

docs(adr): document server authoritative timers
```

Before each commit:

* Run relevant tests.
* Run lint/format.
* Review the diff.
* Ensure no secrets or generated junk are included.

---

# 55. DOCKER REQUIREMENT

All development and testing must continue to support Docker and Docker Compose.

Do not introduce dependencies that cannot run through the existing Docker development environment without documenting and implementing the required changes.

Run relevant tests inside the supported development environment.

Do not silently move the project back to C:.

Respect the existing E: drive requirement from `agents.md`.

---

# 56. IMPLEMENTATION ORDER

Follow this exact high-level order unless repository analysis reveals a strong reason to change it:

1. Inspect existing architecture.
2. Create implementation plan.
3. Centralize configuration.
4. Add/adjust Game State and State Machine.
5. Implement authoritative timers.
6. Implement Hakem timeout.
7. Implement card timeout with lowest legal card.
8. Implement card dealing animation.
9. Implement sequential card presentation.
10. Implement Trick winner state.
11. Implement card collection animation.
12. Implement Round winner display.
13. Implement configurable Round count.
14. Implement persistent authentication/session.
15. Implement reconnection.
16. Implement AI takeover.
17. Integrate SVG card assets.
18. Implement arc hand.
19. Implement tap interaction.
20. Implement drag/drop.
21. Implement drag cancellation.
22. Implement chat configuration.
23. Implement emoji picker.
24. Implement unread badge.
25. Add Unit Tests.
26. Add Integration Tests.
27. Add E2E Tests.
28. Run full regression suite.
29. Update documentation.
30. Update ADRs.
31. Update Lessons Learned.
32. Review Git history.
33. Verify Docker Compose.
34. Complete final acceptance checklist.

Do not skip testing between these stages.

---

# 57. FINAL ACCEPTANCE CRITERIA

This implementation is complete only when all of the following are true:

* Card dealing is animated.
* Each player sees only their own cards.
* Opponent cards remain hidden.
* Card back asset works.
* Hakem has a configurable Trump-selection timer.
* Hakem timer is visually displayed.
* Trump is automatically selected when timeout occurs.
* Card selection timeout is configurable.
* Fast/Medium/Slow settings work.
* Timeout card selection chooses the lowest LEGAL card.
* No timeout selection is random.
* Card play presentation is visually understandable.
* Card animation duration is configurable.
* Trick winner is clearly identified.
* Winner display duration is configurable.
* Cards animate toward the Trick winner.
* Trump is always visible.
* Previous Round winners are visible.
* Match score is visible.
* Match can be configured for different Round counts.
* Refresh does not log the user out.
* Active game state can be restored.
* WebSocket reconnect works.
* Disconnected players have a configurable grace period.
* AI takeover works after grace period.
* AI does not gain unauthorized hidden information.
* Chat can be enabled/disabled by Room creator.
* Disabled Chat is completely hidden.
* Emoji picker works.
* Unread Chat badge works.
* SVG card assets are integrated.
* Card hand uses an arc layout.
* Tap once selects.
* Tap twice plays.
* Drag and drop works.
* Drag cancellation works.
* Cards return smoothly to their original position after cancelled drag.
* Unit tests pass.
* Integration tests pass.
* E2E tests pass.
* Existing regression tests pass.
* Docker Compose works.
* Documentation is updated.
* ADRs are updated.
* Lessons Learned are documented.
* Every coherent change has an appropriate Git commit.
* No gameplay values are scattered as magic numbers.
* All gameplay and timing values are configurable.
* No project data is intentionally stored on C:.
* The application remains server-authoritative.

---

# START IMPLEMENTATION

Start immediately.

Do not ask for confirmation between phases.

First inspect the repository and existing implementation.

Then create/update the phased implementation plan.

Then implement Phase 1.

After Phase 1 is tested, committed, and verified, continue to the next phase.

Follow `AGENTS.md` without exception.
