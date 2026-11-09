# Tungi ekspress / Ночной экспресс / Night Express — Game Design Document

## 1. Pitch
A **text-only interactive mystery** played in the terminal. You are the conductor of carriage 7 on an overnight train
across the steppe. At 22:40 a passenger vanishes. You have until the train arrives at 06:00 — every action costs minutes,
your boss still expects tea and tickets, and someone on board is watching you.

| | |
|---|---|
| Genre | Interactive fiction / "choose your own adventure" mystery with a light time-management simulation |
| Platform | Terminal (macOS / Linux / Windows), single binary |
| Tech | Go (standard library only), story graph + texts embedded with `embed`, `go test` |
| Audience | 14+, fans of detective stories and text adventures; one run 15–25 min; 8 endings encourage replays |
| Languages | Uzbek (Latin), Russian, English (`--lang`, or picked on the title screen) |
| Difficulty | Built into the clock: you cannot see everything in one night; "Hard" mode removes clue hints |

## 2. Setting & characters
Autumn 1979 on train No. 17 "Southern Star" (fictional state railway, no politics), from the river city **Oltinsoy**
(departs 22:00) through the steppe to the desert terminus **Qumtepa** (arrives 06:00), with one long stop at the
**Saksovul junction** 04:00–04:20.

- **You** — conductor of carriage 7 (gender-neutral second person).
- **Professor Arslan Nuriev** — geologist, compartment 4; the one who vanishes. Carries a survey of a huge
  underground freshwater reservoir under the desert.
- **Lola Karimova** — 24, medical student, also in compartment 4; nervous, hides something (she is the professor's
  estranged daughter, travelling to reconcile).
- **Valentin Rudin, "the man in grey"** — compartment 2; says he is "from the ministry". Actually works for a land
  syndicate that wants the survey buried so they can buy the desert cheaply. Antagonist.
- **Gena Sorokin** — compartment 5, card-playing "sewing machine salesman" with debts; paid by Rudin to distract you.
  Red herring who can become an ally.
- **Halima-buvi** — compartment 6, elderly woman travelling with a hen in a basket; sees everything; kindness unlocks her help.
- **Bahodir** — chef of the dining car; gossip, coffee, a hiding place; the dining car closes at 01:00.
- **Sherzod** — 11-year-old stowaway hiding in the baggage car; saw the professor go there.
- **Chief conductor Matvey** — your boss; inspects at 23:00 and 05:00, cares about tea, tickets and order.

**The truth:** at 22:30 the professor caught Rudin searching his luggage. He faked a disappearance (opened the window,
left his briefcase as a decoy), slipped the real map into Halima-buvi's hen basket while she slept, and hid in the
locked baggage car. Rudin's associates wait at Saksovul junction to "escort" the professor off the train.

## 3. Mechanics
- **Clock:** 22:00 → 06:00 (480 minutes), in 10-minute steps. Every choice shows its cost (10–40 min; walking to a
  compartment and back costs 10 + 10, a conversation 10, searching or a stake-out 20, Gena's card game 40). Scenes have
  time windows (Rudin dines 23:30–00:45; dining car open until 01:00; the decoy briefcase can be studied until the
  01:30 blackout; junction 04:00; chief's rounds 23:00 and 05:00). Reaching 06:00 jumps to the arrival and sets the
  clock to exactly 06:00.
- **Stats:** Trust 0–10 (passengers' trust; it only ever rises), Suspicion 0–10 (how much Rudin suspects you; at ≥ 5 he
  sounds you out, at ≥ 7 he threatens, at 10 he attacks unless Gena is your ally), Duty 0–10 (chief's rating; at 0 you
  are relieved of duty), plus money (roubles from Gena's card game, Rudin's "gift" and the E3 deal; shown in status, it
  gates nothing). A hidden **Evidence** counter (0–4) counts key clues found.
- **Clues (flags):** torn_ticket, window_latch, coal_dust, briefcase_decoy, hen_feather_map, rudin_telegram,
  gena_payment, sherzod_witness, lola_photo, baggage_key (10). Key clues needed for the best ending: any 3 of
  {rudin_telegram, gena_payment, briefcase_decoy, sherzod_witness} (Evidence ≥ 3).
- **How each key clue is earned:** the decoy map — study the briefcase before the 01:30 blackout takes it; the
  telegram — search compartment 2 while Rudin dines (23:30–00:45) or in the 01:30 blackout; Gena's payment — play his
  card game (40 min, Duty −1), then confront him; Sherzod — hear Bahodir's gossip (and get bread) before 01:00, then
  listen at the baggage car door.
- **Duties:** tea round (by 23:30), ticket check (by 00:30), coal for the samovar, the 05:00 inspection. Skipping them
  lowers Duty; doing them costs time but raises Trust/Duty and sometimes yields clues (passengers talk while you pour tea).
- **Hub structure:** the corridor of carriage 7 is a hub (doors to compartments 2, 4, 5, 6, your staff room, the
  vestibule toward the dining car, the vestibule toward the baggage car). Each of those locations is a small hub too.
  Timed scenes interrupt on the next entry into any hub after their time comes (never in the middle of a scene),
  each fires at most once, and they are checked in a fixed priority order (Duty 0, Rudin's attack, Saksovul, the 05:00
  inspection, then the rest).
- **Journal:** command `j` lists discovered clues and notes; `s` shows status; `save`/`load`; `lang`; `q` quits.

## 4. Story structure (≈ 45–60 nodes)
1. **Prologue (22:00–22:40):** departure, meet passengers while seating them (choices shape Trust and first
   impressions), the chief's warning.
2. **The disappearance (22:40):** Lola knocks: the professor is gone. Investigate the compartment (window latch,
   briefcase), calm Lola, report to the chief (Duty +, Suspicion + because Rudin overhears) or keep quiet.
3. **Open investigation (22:50–04:00):** free exploration of the hub; interviews; duties; timed interruptions
   (chief's round, Gena's card game invitation, lights failure in which someone searches compartment 4, Rudin's
   friendly-then-threatening chat when Suspicion ≥ 5, finding Sherzod, getting the baggage-car key from the chief or
   Bahodir, Halima-buvi's hen basket).
4. **Saksovul junction (04:00–04:20):** climax — Rudin's men board; where the professor is, who you trust and what
   evidence you hold decide what happens. Never found → taken. Left in the baggage car → safe only if you stand at the
   door with the chief behind you (he was told, and Duty ≥ 6) or Gena (ally) starts a row on the platform. Hidden in
   your staff room → safe if Suspicion < 5; behind Halima-buvi's shawl (needs the hen map) → safe if Suspicion < 7;
   otherwise Rudin followed you and only Gena or the chief can stop him. You can also sell the hiding place (E3).
   If he is taken but you still hold the map, the night goes on (secret E8 is still open).
5. **Dawn (04:20–06:00):** consequences, final inspection, arrival and the militia at Qumtepa → ending.

## 5. Endings (8)
| # | Ending | Condition |
|---|---|---|
| E1 | **Daybreak Witness** (best) | professor safe at arrival + ≥ 3 key clues shown to the militia + Lola reunited with the professor |
| E2 | **Quiet Rescue** | professor safe, but not enough evidence (fewer than 3 key clues, or no reunion — without his daughter beside him the professor will not testify) — Rudin walks away |
| E3 | **The Deal** | you sold the professor's hiding place to Rudin for money |
| E4 | **Wrong Suspect** | you accused Lola or Gena to the militia/chief and they are taken away |
| E5 | **Junction Night** | the professor is taken off the train at Saksovul |
| E6 | **Off the Rails** | Suspicion 10 while alone with Rudin — you wake up in the steppe at dawn |
| E7 | **Relieved of Duty** | Duty reaches 0 — locked in the staff room; the night ends without you |
| E8 | **Keeper of the Water** (secret) | you found the map in the hen basket, kept it (did not give it back to the professor) and, having heard from Bahodir that a reporter meets the train, gave it to her at Qumtepa — whether or not the professor got through Saksovul |
Each ending: 120–200 words, a title card, and a summary (time used, clues found x/10, endings discovered so far — stored
in `~/.night-express/progress.json`).

## 6. Presentation
- ANSI colour (respects `NO_COLOR`, off when output is not a terminal), typewriter effect (Enter shows the rest of
  the text at once — the standard library cannot read single key presses without raw terminal mode; `--fast` turns it
  off), a one-line status bar (🕙 time · Trust · Duty · Suspicion), numbered choices with their time cost, text wrapped
  to `$COLUMNS` (default 80).
- Title screen with ASCII-art train, language and difficulty choice, "Continue" when a save exists.
- `--script "1,3,2,..."` plays a fixed sequence non-interactively (used by tests and demos); `--seed` for the few random
  beats. The random beats are cosmetic only (an ambient line under the corridor text), so every choice is deterministic
  and the solver can be exhaustive; scripts default to seed 1979, so transcripts are reproducible.
- Hard mode (title menu or `--hard`): the journal lists clues without their hints and without the "evidence" marker.

## 7. Testing
- Graph validation: all choice targets exist, every node is reachable, no dead ends (every non-ending node has at least
  one available choice in every reachable state), every ending reachable — via an exhaustive state-space search over
  (node, minutes, stats, flags) with memoisation. To keep it exact yet fast (~8 s, ~0.5 GB, ~5.6 M canonical states
  covering ~52 M concrete ones), the search is layered by minute (time only moves forward), treats Duty × Suspicion as
  an 11×11 set per state, and merges states only when they agree on everything that can still matter (flags no
  condition can read any more, events past their window and monotone stats above their highest threshold are
  dropped). A separate test checks that these reductions change nothing a player could observe.
- "The clock matters": a naive bot that tries every option in the order offered never reaches E1, and none of 3,000
  seeded random nights do; the exhaustive search finds E1 in only ~40 of ~5.6 M canonical states.
- Localisation: every text key exists in uz/ru/en with matching `{placeholders}`.
- Engine unit tests: conditions, effects, clamping, time windows, timed interruptions, save/load round trip.
- Scripted playthroughs that reach specific endings (at least E1, E5, E7 and E8) in each language.
