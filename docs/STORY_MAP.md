# Night Express — Story Map

> **SPOILER WARNING.** This file reveals the whole mystery, every hidden condition and all eight endings. Play first.

Generated from `story/graph.json` (edges, costs and conditions are exact; summaries are hand-written). Time runs 22:00 → 06:00 in 10-minute steps; every choice costs 10–40 minutes. Hubs (`[hub]`) are where timed events interrupt. Effects use `+flag`, `stat±n`.

## The truth in one paragraph

At 22:30 Professor Nuriev caught Rudin (a land syndicate's man posing as a ministry official) going through his briefcase. He drew a fake map, left it in the briefcase, pushed the window latch up with a pencil to fake a jump, hid the real survey (a vast fresh-water lens under the Qumtepa desert) under Halima-buvi's hen while she slept, and slipped in his socks into the baggage car when Bahodir left it open for a minute. Sherzod, a stowaway boy living there, saw him. Rudin paid Gena to keep the conductor busy; three men and a car wait at Saksovul (04:00) to take the professor. Lola, in the same compartment, is the professor's estranged daughter.

## Timed events (fire once, at the next hub visit, in this priority order)

| Event | From | Until | Condition | Scene |
|---|---|---|---|---|
| duty_zero | 22:00 | — | duty<=0 | `e7_relieved` |
| rudin_attack | 22:00 | — | suspicion>=10 | `ev_rudin_attack` |
| junction | 04:00 | — | — | `ev_junction` |
| inspection | 05:00 | — | — | `ev_inspection` |
| chief_round | 23:00 | 01:00 | — | `ev_chief_round` |
| tea_late | 23:30 | — | !tea_done | `ev_tea_late` |
| tickets_late | 00:30 | — | !tickets_done | `ev_tickets_late` |
| lights | 01:30 | 02:30 | — | `ev_lights` |
| gena_cards | 23:45 | 01:30 | !gena_payment | `ev_gena_cards` |
| hen_restless | 02:30 | 03:45 | halima_talked & !hen_feather_map & trust>=4 | `ev_hen_restless` |
| rudin_chat | 22:50 | 04:00 | suspicion>=5 | `ev_rudin_chat` |
| rudin_threat | 22:50 | 04:00 | suspicion>=7 | `ev_rudin_threat` |

Reaching 06:00 from any non-final node jumps to `arrival` and sets the clock to 06:00.

## Nodes and edges

### `p_departure` [scene]

22:00, Oltinsoy platform. Five passengers board; you can help only one first.

- `help_halima` → `p_halima` (10 min) — +halima_kind, trust+1
- `help_lola` → `p_lola` (10 min) — +lola_kind, trust+1
- `check_grey` → `p_rudin` (10 min) — +met_rudin

### `p_halima` [scene]

Halima-buvi and Malika the hen. She sleeps soundly after her heart drops; the hen 'keeps watch'.

- `next` → `p_chief` (10 min)

### `p_lola` [scene]

Lola carries a suitcase of books, asks whether the bearded man goes to Qumtepa; a photo corner in her textbook.

- `next` → `p_chief` (10 min)

### `p_rudin` [scene]

Rudin ('Ministry') and Gena the salesman; Rudin asks if the professor sleeps soundly.

- `next` → `p_chief` (10 min)

### `p_chief` [scene]

Chief Matvey's rules: tea by 23:30, tickets by 00:30, coal for 05:00, Saksovul at 04:00; keep the ministry man happy.

- `yes` → `p_professor` (10 min) — duty+1
- `ask` → `p_professor` (10 min) — +note_ministry

### `p_professor` [scene]

Tea for Professor Nuriev, who asks who keeps the baggage-car key and how long Saksovul lasts.

- `goodnight` → `d_knock` (10 min)
- `ask_baggage` → `d_knock` (10 min) — +note_baggage

### `d_knock` [scene]

22:40: Lola - the professor is gone, window open; compartment 2's door clicks shut.

- `go` → `d_comp4` (10 min)

### `d_comp4` [scene]

Compartment 4: window (pencil-pushed latch, ticket scrap '...02'), the decoy map in the briefcase, calm Lola, report or keep quiet.

- `latch` → `d_comp4` (10 min) — if !window_latch — +window_latch
- `briefcase` → `d_comp4` (10 min) — if !briefcase_decoy — +briefcase_decoy, evidence+1
- `calm` → `d_comp4` (10 min) — if !lola_calmed — +lola_calmed, trust+1
- `report` → `d_chief_report` (10 min) — if !chief_informed — +chief_informed, duty+1, suspicion+2
- `quiet` → `corridor` (10 min)

### `d_chief_report` [scene]

Matvey is told; Rudin overhears through his open door (Suspicion +2).

- `back` → `corridor` (10 min)

### `corridor` [hub]

HUB. Doors to all locations, the tea and ticket rounds (time-limited), waiting. Timed events interrupt here.

- `tea` → `duty_tea` (20 min) — if !tea_done & <23:30 — +tea_done, duty+1, suspicion-1
- `tickets` → `duty_tickets` (20 min) — if !tickets_done & <00:30 — +tickets_done, duty+1
- `c2` → `c2` (10 min)
- `c4` → `c4` (10 min)
- `c5` → `c5` (10 min)
- `c6` → `c6` (10 min)
- `staff` → `staff` (10 min)
- `dining` → `dining` (10 min)
- `rear` → `rear` (10 min)
- `wait` → `corridor` (30 min)

### `c2` [hub]

Rudin's compartment. Talk (Suspicion +1, repeatable); empty 23:30-00:45 while he dines - search it.

- `talk` → `c2` (10 min) — if !prof_taken & !(>=23:30 & <00:45) — +rudin_talked, suspicion+1
- `search` → `c2_search` (20 min) — if >=23:30 & <00:45 & !rudin_telegram
- `back` → `corridor` (10 min)

### `c2_search` [scene]

KEY CLUE rudin_telegram: 'Saksovul 04:00, three men and car, bring geologist with papers'. Copy it, or pocket it (Suspicion +2).

On enter: +rudin_telegram, evidence+1

- `copy` → `c2` (10 min)
- `pocket` → `c2` (10 min) — suspicion+2

### `c4` [hub]

Lola. Talk (photo needs Trust >= 5), re-examine the window, study the briefcase (before 01:30), bring her to her father.

- `talk_open` → `c4` (10 min) — if !lola_photo & trust>=5 — +lola_photo, +lola_talked, trust+1
- `talk_first` → `c4` (10 min) — if !lola_talked & !lola_photo & trust<5 — +lola_talked, trust+1
- `talk_again` → `c4` (10 min) — if lola_talked & !lola_photo & trust<5
- `window` → `c4` (10 min) — +window_latch
- `briefcase` → `c4` (10 min) — if <01:30 & !briefcase_decoy — +briefcase_decoy, evidence+1
- `fetch` → `reunion` (20 min) — if prof_knows_lola & !reunited & !prof_taken
- `back` → `corridor` (10 min)

### `c5` [hub]

Gena. His patter hints at 'a generous friend'; after the card game you can confront him; with Trust >= 5 he becomes an ally.

- `talk` → `c5` (10 min) — if !gena_payment & !gena_cards — +gena_talked
- `confront` → `gena_confront` (10 min) — if gena_cards & !gena_payment — +gena_talked
- `ally` → `c5` (10 min) — if gena_payment & !gena_ally & trust>=5 — +gena_ally
- `ally_no` → `c5` (10 min) — if gena_payment & !gena_ally & trust<5
- `back` → `corridor` (10 min)

### `gena_confront` [scene]

KEY CLUE gena_payment: Rudin paid 50 roubles 'for the distraction'.

On enter: +gena_payment, evidence+1

- `ally` → `c5` (10 min) — if trust>=5 — +gena_ally
- `leave` → `c5` (10 min)

### `c6` [hub]

Halima-buvi: she saw the professor walk to the back in socks; Malika sits 'heavy, like on eggs'. Asking about the basket needs kindness or Trust >= 6.

- `talk` → `c6` (10 min) — if !halima_talked — +halima_talked, trust+1
- `hen` → `halima_basket` (10 min) — if halima_talked & !hen_feather_map & (halima_kind | trust>=6)
- `hen_no` → `c6` (10 min) — if halima_talked & !hen_feather_map & !halima_kind & trust<6
- `back` → `corridor` (10 min)

### `halima_basket` [scene]

The real survey map under the hen (hen_feather_map): a sea of fresh water under the desert.

On enter: +hen_feather_map

- `keep` → `c6` (10 min)

### `staff` [hub]

Your staff room: the chief's orders (Rudin alights at Saksovul 'with a companion'); sleeping costs Duty 2.

- `orders` → `staff` (10 min) — +read_orders
- `sleep` → `corridor` (40 min) — duty-2
- `back` → `corridor` (10 min)

### `dining` [hub]

Bahodir (until 01:00): baggage door 'felt heavier' at 22:35, the reporter's telegram, Rudin dines 23:30-00:45, bread; spare key with Trust >= 5.

- `chat` → `dining` (10 min) — if <01:00 & !bahodir_talked — +bahodir_talked
- `key` → `dining` (10 min) — if <01:00 & bahodir_talked & !baggage_key & trust>=5 — +baggage_key
- `key_no` → `dining` (10 min) — if <01:00 & bahodir_talked & !baggage_key & trust<5
- `back` → `corridor` (10 min)

### `rear` [hub]

Rear vestibule: coal (coal_dust sock prints, Duty +1), the hidden child (needs Bahodir's bread), the baggage car (needs a key).

- `coal` → `rear` (20 min) — if !coal_done — +coal_done, +coal_dust, duty+1
- `listen_bread` → `sherzod` (20 min) — if bahodir_talked & !sherzod_witness
- `listen` → `rear` (10 min) — if !bahodir_talked & !sherzod_witness — +sherzod_heard
- `open` → `baggage` (10 min) — if baggage_key & !prof_found
- `visit` → `baggage` (10 min) — if prof_found & !hid_staff & !hid_halima & !prof_taken
- `back` → `corridor` (10 min)

### `sherzod` [scene]

KEY CLUE sherzod_witness: the boy saw Rudin leave compartment 4 and the professor slip into the baggage car.

On enter: +sherzod_witness, evidence+1, trust+1

- `thank` → `rear` (10 min)

### `baggage` [scene]

The professor among the melons. Hear his story, tell him about Lola, give back the map, hide him (staff room / Halima) or leave him.

On enter: +prof_found

- `ask` → `prof_talk` (10 min)
- `tell_lola` → `baggage` (10 min) — if lola_photo & !prof_knows_lola — +prof_knows_lola
- `give_map` → `baggage` (10 min) — if hen_feather_map & !map_returned — +map_returned, trust+1
- `hide_staff` → `corridor` (20 min) — if !prof_safe — +hid_staff, suspicion+1
- `hide_halima` → `corridor` (20 min) — if hen_feather_map & !prof_safe — +hid_halima
- `leave` → `corridor` (10 min)

### `prof_talk` [scene]

The truth: the decoy, the window trick, the map under the hen, the water; he will testify only for a daughter.

On enter: +prof_told

- `back` → `baggage` (10 min)

### `reunion` [scene]

Father and daughter meet (reunited).

On enter: +reunited, trust+1

- `leave` → `corridor` (10 min)

### `duty_tea` [scene]

Tea round: passengers talk; Rudin pushes the 'he jumped' story.

- `next` → `corridor` (10 min)

### `duty_tickets` [scene]

Ticket round: Gena's and Rudin's tickets were bought together; Rudin's has a torn corner - matches the scrap if you examined the window (torn_ticket).

- `match` → `corridor` (10 min) — if window_latch — +torn_ticket
- `finish` → `corridor` (10 min)

### `ev_chief_round` [scene]

23:00 chief's round: report the disappearance; with Duty >= 6 he lends you the baggage-car key.

- `report` → `ev_chief_round` (10 min) — if !chief_informed — +chief_informed, duty+1, suspicion+2
- `key` → `corridor` (10 min) — if chief_informed & !baggage_key & duty>=6 — +baggage_key
- `key_no` → `corridor` (10 min) — if chief_informed & !baggage_key & duty<6
- `fine` → `corridor` (10 min)

### `ev_tea_late` [scene]

23:30 if no tea: Duty -2.

On enter: duty-2

- `sorry` → `corridor` (10 min)

### `ev_tickets_late` [scene]

00:30 if no tickets: Duty -2.

On enter: duty-2

- `sorry` → `corridor` (10 min)

### `ev_gena_cards` [scene]

23:45-01:30 Gena's card game: 40 minutes and Duty -1, but you see the ministry envelope (unlocks the confrontation).

- `play` → `corridor` (40 min) — +gena_cards, money+10, duty-1
- `decline` → `corridor` (10 min)

### `ev_lights` [scene]

01:30 blackout: someone takes the briefcase. Guard compartment 4, search compartment 2 in the dark (telegram), or stay with Halima.

- `guard` → `corridor` (10 min) — +saw_searcher, trust+1, suspicion+1
- `search` → `corridor` (20 min) — if !rudin_telegram — +rudin_telegram, evidence+1
- `stay` → `corridor` (10 min) — trust+1

### `ev_hen_restless` [scene]

02:30-03:45, if you sat with Halima but never asked: she fetches you to look at the basket.

- `go` → `halima_basket` (10 min)
- `later` → `corridor` (10 min)

### `ev_rudin_chat` [scene]

Suspicion >= 5: Rudin's friendly chat - play dumb, take his 20 roubles, push him, or sell the professor (E3).

- `dumb` → `corridor` (10 min) — suspicion-2
- `gift` → `corridor` (10 min) — money+20, suspicion-3, +took_bribe
- `push` → `corridor` (10 min) — suspicion+2, +rudin_water
- `sell` → `e3_deal` (10 min) — if prof_found — money+100

### `ev_rudin_threat` [scene]

Suspicion >= 7: Rudin's threat - promise, defy, or sell (E3).

- `promise` → `corridor` (10 min) — suspicion-2, +promised_rudin
- `defy` → `corridor` (10 min) — suspicion+2
- `sell` → `e3_deal` (10 min) — if prof_found — money+100

### `ev_rudin_attack` [scene]

Suspicion 10: Rudin at the open door. Only Gena (ally) can save you; otherwise E6.

- `shout` → `corridor` (10 min) — if gena_ally — suspicion-3
- `fight` → `e6_rails` (10 min) — if !gena_ally

### `ev_junction` [scene]

04:00 Saksovul: three men board. Outcome depends on where the professor is, the chief's backing, Gena and Suspicion.

- `follow` → `j_taken` (10 min) — if !prof_found
- `stand_ok` → `j_safe` (20 min) — if prof_found & !hid_staff & !hid_halima & chief_informed & duty>=6 — +chief_blocked
- `stand_bad` → `j_taken` (20 min) — if prof_found & !hid_staff & !hid_halima & (!chief_informed | duty<6)
- `gena_row` → `j_safe` (20 min) — if prof_found & !hid_staff & !hid_halima & gena_ally — +gena_row
- `let_ok` → `j_safe` (20 min) — if (hid_staff & suspicion<5 | hid_halima & suspicion<7)
- `let_bad` → `j_followed` (20 min) — if (hid_staff | hid_halima) & !((hid_staff & suspicion<5 | hid_halima & suspicion<7))
- `sell` → `e3_deal` (10 min) — if prof_found — money+100

### `j_followed` [scene]

Rudin followed you to the hiding place: allies (Gena, or the chief with Duty >= 6) or nothing.

- `allies` → `j_safe` (10 min) — if (gena_ally | chief_informed & duty>=6) — +allies_came
- `alone` → `j_taken` (10 min) — if !((gena_ally | chief_informed & duty>=6))

### `j_safe` [scene]

The men leave empty-handed; Rudin stays aboard (prof_safe).

On enter: +prof_safe

- `on` → `corridor` (10 min)

### `j_taken` [scene]

The professor is taken. With the map still in your tunic the night goes on (prof_taken), else E5.

- `keep_map` → `corridor` (10 min) — if hen_feather_map & !map_returned — +prof_taken
- `end` → `e5_junction` (10 min)

### `ev_inspection` [scene]

05:00 final inspection: hot samovar (coal done) Duty +1, else -1.

- `report_ok` → `corridor` (10 min) — if coal_done — duty+1
- `report_cold` → `corridor` (10 min) — if !coal_done — duty-1

### `arrival` [scene]

06:00 Qumtepa (reaching 06:00 forces this).

- `door` → `platform` (10 min)

### `platform` [scene]

The militia captain, Rudin leaving, and - if Bahodir told you - the reporter. Your last choice.

- `evidence_win` → `e1_daybreak` (10 min) — if prof_safe & reunited & evidence>=3
- `evidence` → `e2_quiet` (10 min) — if prof_safe & !(reunited & evidence>=3)
- `reporter` → `e8_keeper` (10 min) — if hen_feather_map & !map_returned & bahodir_talked
- `accuse_lola` → `e4_wrong` (10 min)
- `accuse_gena` → `e4_wrong` (10 min)
- `silent` → `e5_junction` (10 min) — if !prof_safe

### `e1_daybreak` [ending]

E1 Daybreak Witness: professor safe + reunited with Lola + Evidence >= 3 -> Rudin detained.

On enter: ending E1


### `e2_quiet` [ending]

E2 Quiet Rescue: professor safe but the case is thin (no reunion or < 3 key clues); Rudin walks away.

On enter: ending E2


### `e3_deal` [ending]

E3 The Deal: you sold the hiding place for 100 roubles.

On enter: ending E3


### `e4_wrong` [ending]

E4 Wrong Suspect: you accused Lola or Gena.

On enter: ending E4


### `e5_junction` [ending]

E5 Junction Night: the professor is taken at Saksovul.

On enter: ending E5


### `e6_rails` [ending]

E6 Off the Rails: Suspicion 10, alone with Rudin.

On enter: ending E6


### `e7_relieved` [ending]

E7 Relieved of Duty: Duty 0.

On enter: ending E7


### `e8_keeper` [ending]

E8 Keeper of the Water (secret): map kept + reporter known -> the map goes to the newspaper.

On enter: ending E8


## Tested routes

See the README spoiler section for the exact `--script` sequences (E1, E5, E7, E8) and `engine/playthrough_test.go` for the same routes as choice ids.

