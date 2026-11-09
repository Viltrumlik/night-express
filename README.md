# Night Express · Tungi ekspress · Ночной экспресс

![Go: stdlib only](https://img.shields.io/badge/Go-stdlib%20only-00ADD8?logo=go&logoColor=white) ![endings: 8](https://img.shields.io/badge/endings-8-8B5CF6) ![tests: passing](https://img.shields.io/badge/tests-passing-2ea44f) ![languages: uz · ru · en](https://img.shields.io/badge/languages-uz%20%C2%B7%20ru%20%C2%B7%20en-orange) ![license: proprietary](https://img.shields.io/badge/license-proprietary-lightgrey)

A text-only interactive-fiction mystery for the terminal. You are the conductor of carriage 7 on an overnight train across the steppe in the autumn of 1979. At 22:40 a passenger vanishes, and you have until the train pulls into Qumtepa at 06:00 to find out what happened. Every action costs minutes, the chief conductor still wants his tea and tickets, and someone on board is watching how curious you are. The night is shorter than everything there is to do, so what you read and what you skip both matter.

- **Three languages, written as literature:** Uzbek (Latin script), Russian and English, switchable at any moment.
- **A real clock:** 22:00 → 06:00 in 10-minute steps. Rooms open and close, and 12 timed interruptions fire whether you are ready or not (the chief's rounds, a card game, a blackout, the Saksovul junction at 04:00, the 05:00 inspection…).
- **50 scenes and 8 endings,** one of them secret. The best ending needs attentive reading and a plan; trying everything in order will not get you there.
- **Stats that bite:** Trust, Duty and Suspicion (0–10), money, and a journal of 10 clues and dozens of notes.
- **Save/Continue** (one save slot), endings collection across runs, a hard mode without clue hints, colour that respects `NO_COLOR`, a skippable typewriter effect, and word-wrap to your terminal width.
- **Single static binary:** Go standard library only, with the story embedded.

## What it looks like

The opening of a night in English (`--fast`, colour off). Type a number and press Enter; every choice shows
what it costs on the clock.

```text
Oltinsoy, 22:00. The river breathes cold mist across the platform, and train
No. 17, the "Southern Star", hisses like a samovar about to boil. You stand
at the door of carriage 7 with your lamp and your punch, cap straight,
collar buttoned, the way Chief Conductor Matvey likes it.

Your passengers come out of the mist one by one. An old woman with a wicker
basket that clucks. A young woman dragging a suitcase heavier than she is,
who keeps glancing at the lit windows. A man in a grey coat with a thin
leather folder, followed by a loud fellow juggling two sample cases. Last, a
tall grey-bearded man who holds his briefcase against his chest like a
child.

The whistle blows. You have one pair of hands.

🕙 22:00 · Trust 3 · Duty 5 · Suspicion 1
 1. Help the old woman lift her basket aboard (10 min)
 2. Take the young woman's heavy suitcase (10 min)
 3. Check the tickets of the man in grey and his loud companion (10 min)
> _
```

## Qisqacha · Кратко

**🇺🇿 O'zbekcha.** Terminalda o'ynaladigan matnli detektiv. 1979-yil kuzi, 17-raqamli «Janubiy yulduz» poyezdi. Siz 7-vagon provodnikisiz; soat 22:40 da yo'lovchi, geolog professor g'oyib bo'ladi. Poyezd 06:00 da Qumtepaga yetib keladi, har bir harakat esa daqiqalarni yeydi. 50 sahna, 10 ta iz, 8 ta yakun (bittasi sirli). O'zbek tilida boshlash: `go run ./cmd/nightexpress --lang uz --new`.

**🇷🇺 Русский.** Текстовый детектив для терминала. Осень 1979 года, поезд № 17 «Южная звезда». Вы — проводник седьмого вагона; в 22:40 пропадает пассажир, профессор-геолог. Поезд прибывает в Кумтепу в 06:00, и каждое действие стоит минут. 50 сцен, 10 улик, 8 концовок (одна тайная). Запуск на русском: `go run ./cmd/nightexpress --lang ru --new`.

## The story so far (no spoilers)

Train No. 17, the "Southern Star", leaves the river town of Oltinsoy at ten at night. In your carriage: Professor Arslan Nuriev, a geologist who holds his briefcase like a child. Lola, a medical student who won't look at him. Valentin Rudin, "from the ministry", who asks whether the professor sleeps soundly. Gena, a salesman with debts and a surprisingly fat wallet. And Halima-buvi, an old woman travelling with a hen in a basket, who says she sleeps like a stone. Forty minutes after departure, Lola hammers on your door: the professor is gone, and the window is open.

## How to play

Read the scene, then type the number of a choice and press Enter. Each choice shows its cost in minutes, and the train arrives at 06:00 no matter what.

| Command | What it does |
|---|---|
| `1`, `2`, … | take that choice |
| `s` | status: time left, Trust/Duty/Suspicion, money, clues, evidence |
| `j` | journal of clues and notes (hard mode hides the hints) |
| `save` / `load` | save the night / load the saved night (`~/.night-express/save.json`) |
| `lang` (or `lang uz`, `lang ru`, `lang en`) | switch language mid-game |
| `h` | help |
| `q` | quit (use `save` first if you want to continue later) |

While text is printing, press Enter to show it all at once.

Command-line flags:

| Flag | Meaning |
|---|---|
| `--lang uz\|ru\|en` | language (otherwise chosen on the title screen) |
| `--new` | skip the title menu and start a new night |
| `--hard` | hard mode: the journal lists clues without hints |
| `--fast` | no typewriter effect |
| `--script "1,3,2,…"` | play a fixed comma-separated list of inputs non-interactively and print the transcript. Commands such as `j` or `lang ru` are allowed as tokens. Exits 0 if an ending is reached, non-zero otherwise. |
| `--seed N` | seed for the few random, purely cosmetic beats (scripts default to 1979, so transcripts are reproducible) |

Progress (the endings you have discovered) is kept in `~/.night-express/progress.json`. Set `NIGHT_EXPRESS_HOME` to use another directory.

## Requirements

- Go 1.27 or newer (standard library only, no third-party modules).
- Any terminal on macOS, Linux or Windows. The status line uses one emoji (🕙). Set `COLUMNS` if your shell does not export it (the default width is 80).

## Build and run

```sh
git clone https://github.com/Viltrumlik/night-express.git
cd night-express
go build -o nightexpress ./cmd/nightexpress
./nightexpress                 # title screen: language, difficulty, Continue
./nightexpress --lang uz --new # straight into a new night, in Uzbek
```

Or without building:

```sh
go run ./cmd/nightexpress --lang en --new
```

## Tests

```sh
go vet ./...
go test ./...          # about 10 s; the exhaustive solver uses ~0.5 GB of RAM
go test -short ./...   # skips the exhaustive search and the slow soundness check
```

The suite has 28 top-level tests (52 runs including subtests):

- **Validation.** All choice targets exist, every node and every ending is reachable, and there are no dead ends. This comes from an exhaustive memoised search over every reachable (node, minutes, stats, flags) state: about 5.6 million canonical states, or about 52 million concrete ones. A second test checks that the search's state-merging changes nothing a player could observe.
- **Localisation.** The three languages have identical key sets and identical `{placeholders}`, no empty strings, Uzbek uses only the ASCII apostrophe, and the prose is actually translated.
- **Engine unit tests.** Every condition type, effects and clamping, time windows across midnight, timed events firing once and in priority order, the 06:00 deadline, graph validation errors, and save/load and progress round trips.
- **Playthroughs.** Scripted runs through the same code path as `--script` reach all eight endings in all three languages. 3,000 seeded random nights never get stuck, and the "clock matters" bot shows that trying every option in order does not reach the best ending.

## Project structure

```
cmd/nightexpress/   command-line entry point (flags, exit codes)
ui/                 title screen, game loop, commands, --script mode, wrapping, colour, typewriter
engine/             loading + validation, state, conditions/effects, timed events, save/progress,
                    exhaustive solver (explore.go) and the tests
story/graph.json    the language-independent story graph: nodes, choices, conditions, effects, timed events
story/text/         uz.json, ru.json, en.json: key → text
docs/GDD.md         game design document
docs/STORY_MAP.md   every node and edge (spoilers!)
```

The design document is [docs/GDD.md](docs/GDD.md).

## Spoilers: scripted endings

<details>
<summary><strong>SPOILER WARNING: click to show <code>--script</code> sequences that reach each ending</strong></summary>

Each line plays a whole night non-interactively. Add `--lang uz` or `--lang ru` for the other languages; the numbers are the same. All eight are covered by the tests in every language.

```sh
# E1 Daybreak Witness (best ending)
./nightexpress --lang en --script "2,1,1,1,1,5,1,1,1,1,2,7,1,1,1,2,3,1,1,2,2,7,2,1,2,2,3,2,2,1,8,1,1,8,1,8,8,1,1"
# E2 Quiet Rescue
./nightexpress --lang en --script "2,1,1,1,1,5,1,1,1,1,2,7,1,1,1,2,3,1,1,2,2,7,2,1,2,2,3,8,8,1,1,8,1,8,8,1,1"
# E3 The Deal
./nightexpress --lang en --script "2,1,1,1,1,5,1,1,9,1,8,2,8,3,3,1,8,8,3,8,8,8,8,8,2"
# E4 Wrong Suspect
./nightexpress --lang en --script "1,1,1,1,1,5,2,6,1,1,5,2,5,1,1,1,6,1,1,8,3,8,8,8,8,8,1,1,8,1,8,8,1,2"
# E5 Junction Night
./nightexpress --lang en --script "3,1,1,1,1,5,2,1,1,9,2,9,1,8,3,8,8,8,8,8,1,1"
# E6 Off the Rails
./nightexpress --lang en --script "3,1,1,1,1,4,1,2,3,1,2,2,2,1,2,1,3,2,1,1,1"
# E7 Relieved of Duty
./nightexpress --lang en --script "3,1,2,1,1,5,2,7,2,1,2,6,1"
# E8 Keeper of the Water (secret)
./nightexpress --lang en --script "1,1,1,1,1,5,2,6,1,1,5,2,5,1,1,1,6,1,1,8,3,8,8,8,8,8,1,1,8,1,8,8,1,1"
```

The same routes, written as readable choice ids with comments, are in `engine/playthrough_test.go`. `docs/STORY_MAP.md` lists every hidden condition.

</details>

## More games by Viltrumlik

- [Karvonboshi](https://github.com/Viltrumlik/karvon-bot): Silk Road trading roguelite you play in a Telegram chat
- [Last Shore](https://github.com/Viltrumlik/last-shore): pixel-art survival city-builder on a drying sea (browser)
- [Dispatch 112](https://github.com/Viltrumlik/dispatch-112): real-time emergency-dispatch sim (terminal)
- [Zero to Mars](https://github.com/Viltrumlik/zero-to-mars): 3D founder life-sim Telegram WebApp

## License

Proprietary. Copyright (c) 2026 Viltrumlik. All rights reserved. See [LICENSE](LICENSE).
