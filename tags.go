package main

import "math/rand"

// TagID identifies a systemic tag. Tags are the game's global memory.
// Everything — player, rooms, enemies — accumulates tags that bias
// all future procedural generation and AI behavior.
type TagID int

const (
	// --- Player state tags ---
	TagWounded   TagID = iota // took damage recently
	TagBleeding               // sustained damage, drains HP over time
	TagLoud                   // aggressive movement, alerts enemies
	TagStealthy               // hasn't been hit, careful play
	TagBerserk                // killed many enemies quickly
	TagCursed                 // accumulated bad luck / messy outcomes
	TagOathBound              // made a deal with the dungeon (future)
	TagOvercharged            // too many items, power is unstable

	// --- Environment tags (applied to rooms) ---
	TagScorched   // fire was used here
	TagContested  // heavy combat happened
	TagDebris     // explosions / destruction
	TagBloodSoaked // many deaths happened
	TagQuiet      // room cleared cleanly
	TagDefiled    // poison was used
	TagFrozen     // ice was used

	// --- Entity tags (applied to enemies) ---
	TagAlert      // aware of player, aiming at door
	TagCornered   // low HP, desperate
	TagAggressive // boosted offense
	TagFrenzied   // allies died, going berserk
	TagBloodhound // hunting a bleeding player
	TagShielded   // defensive stance
	TagRetaliate  // just got hit, counterattacking

	SystemTagCount
)

// TagMeta holds display info for visual readability.
type TagMeta struct {
	Name  string
	Tint  [3]uint8 // R,G,B for visual cues
	Decay float32  // seconds until tag fades (0 = permanent until removed)
}

var tagMeta = [SystemTagCount]TagMeta{
	TagWounded:     {Name: "Wounded", Tint: [3]uint8{255, 80, 80}, Decay: 15},
	TagBleeding:    {Name: "Bleeding", Tint: [3]uint8{200, 30, 30}, Decay: 10},
	TagLoud:        {Name: "Loud", Tint: [3]uint8{255, 200, 60}, Decay: 8},
	TagStealthy:    {Name: "Stealthy", Tint: [3]uint8{100, 180, 255}},
	TagBerserk:     {Name: "Berserk", Tint: [3]uint8{255, 60, 200}, Decay: 6},
	TagCursed:      {Name: "Cursed", Tint: [3]uint8{160, 60, 200}},
	TagOathBound:   {Name: "Oath-bound", Tint: [3]uint8{200, 180, 255}},
	TagOvercharged: {Name: "Overcharged", Tint: [3]uint8{255, 255, 100}},

	TagScorched:   {Name: "Scorched", Tint: [3]uint8{255, 140, 40}},
	TagContested:  {Name: "Contested", Tint: [3]uint8{200, 60, 60}},
	TagDebris:     {Name: "Debris", Tint: [3]uint8{160, 140, 100}},
	TagBloodSoaked:{Name: "Blood-soaked", Tint: [3]uint8{180, 20, 20}},
	TagQuiet:      {Name: "Quiet", Tint: [3]uint8{100, 200, 180}},
	TagDefiled:    {Name: "Defiled", Tint: [3]uint8{80, 200, 60}},
	TagFrozen:     {Name: "Frozen", Tint: [3]uint8{100, 180, 255}},

	TagAlert:      {Name: "Alert", Tint: [3]uint8{255, 220, 60}},
	TagCornered:   {Name: "Cornered", Tint: [3]uint8{255, 100, 40}},
	TagAggressive: {Name: "Aggressive", Tint: [3]uint8{255, 60, 60}},
	TagFrenzied:   {Name: "Frenzied", Tint: [3]uint8{255, 40, 100}},
	TagBloodhound: {Name: "Bloodhound", Tint: [3]uint8{200, 30, 30}},
	TagShielded:   {Name: "Shielded", Tint: [3]uint8{80, 160, 255}},
	TagRetaliate:  {Name: "Retaliate", Tint: [3]uint8{255, 180, 40}},
}

// TagSet is the universal tag container. Attached to player, rooms, enemies.
type TagSet struct {
	Active map[TagID]float32 // tag → remaining duration (<=0 means permanent)
}

func NewTagSet() TagSet {
	return TagSet{Active: make(map[TagID]float32)}
}

func (ts *TagSet) Has(t TagID) bool {
	_, ok := ts.Active[t]
	return ok
}

// Apply adds a tag. If it has a default decay, uses that. Otherwise permanent.
func (ts *TagSet) Apply(t TagID) {
	decay := tagMeta[t].Decay
	if decay > 0 {
		// Refresh or extend duration
		if existing, ok := ts.Active[t]; ok && existing > decay {
			return // don't shorten
		}
		ts.Active[t] = decay
	} else {
		ts.Active[t] = 0 // permanent
	}
}

// ApplyDuration adds a tag with explicit duration.
func (ts *TagSet) ApplyDuration(t TagID, dur float32) {
	if existing, ok := ts.Active[t]; ok && existing > dur {
		return
	}
	ts.Active[t] = dur
}

func (ts *TagSet) Remove(t TagID) {
	delete(ts.Active, t)
}

func (ts *TagSet) Clear() {
	for k := range ts.Active {
		delete(ts.Active, k)
	}
}

// Tick decays timed tags. Call once per frame with dt.
func (ts *TagSet) Tick(dt float32) {
	for t, dur := range ts.Active {
		if dur <= 0 {
			continue // permanent
		}
		dur -= dt
		if dur <= 0 {
			delete(ts.Active, t)
		} else {
			ts.Active[t] = dur
		}
	}
}

// Count returns how many tags are currently active.
func (ts *TagSet) Count() int {
	return len(ts.Active)
}

// List returns all active tag IDs.
func (ts *TagSet) List() []TagID {
	out := make([]TagID, 0, len(ts.Active))
	for t := range ts.Active {
		out = append(out, t)
	}
	return out
}

// ---------- PbtA Resolution Engine ----------
// Every significant interaction passes through this.
// Player tags shift the odds.

type Outcome int

const (
	OutcomeClean  Outcome = iota // rare: flawless execution
	OutcomeMessy                 // default: success with consequence
	OutcomeCostly                // failure: reduced effect + penalty
)

// Resolve runs PbtA-style resolution.
// Base odds: 15% clean, 55% messy, 30% costly.
// Player tags shift these odds.
func Resolve(rng *rand.Rand, playerTags *TagSet) Outcome {
	cleanChance := 15
	messyChance := 55
	// costlyChance = 100 - clean - messy

	// Tag modifiers
	if playerTags.Has(TagStealthy) {
		cleanChance += 15
		messyChance += 10
	}
	if playerTags.Has(TagBerserk) {
		cleanChance += 10 // berserk = skilled from killstreak
	}
	if playerTags.Has(TagWounded) {
		cleanChance -= 5
		messyChance -= 10 // more costly outcomes when hurt
	}
	if playerTags.Has(TagCursed) {
		cleanChance -= 10
	}
	if playerTags.Has(TagBleeding) {
		cleanChance -= 5
	}
	if playerTags.Has(TagOvercharged) {
		messyChance += 15 // more messy, fewer clean/costly extremes
		cleanChance -= 5
	}

	// Clamp
	if cleanChance < 5 {
		cleanChance = 5
	}
	if messyChance < 20 {
		messyChance = 20
	}
	total := cleanChance + messyChance
	if total > 95 {
		messyChance = 95 - cleanChance
	}

	roll := rng.Intn(100)
	if roll < cleanChance {
		return OutcomeClean
	}
	if roll < cleanChance+messyChance {
		return OutcomeMessy
	}
	return OutcomeCostly
}

// ---------- Tag Consequence Tables ----------
// When a messy/costly outcome happens, what tags get applied?

// MessyConsequence picks a consequence tag for a messy combat hit.
func MessyConsequence(rng *rand.Rand, playerTags *TagSet, roomTags *TagSet) {
	roll := rng.Intn(100)
	switch {
	case roll < 25:
		playerTags.Apply(TagLoud)
	case roll < 45:
		roomTags.Apply(TagDebris)
	case roll < 60:
		roomTags.Apply(TagContested)
	case roll < 75:
		// Element-specific room tags based on projectile
		if playerTags.Has(TagBerserk) {
			roomTags.Apply(TagBloodSoaked)
		} else {
			roomTags.Apply(TagContested)
		}
	default:
		// No extra consequence this time
	}
}

// CostlyConsequence applies penalties for a costly failure.
func CostlyConsequence(rng *rand.Rand, playerTags *TagSet) {
	roll := rng.Intn(100)
	switch {
	case roll < 35:
		playerTags.Apply(TagWounded)
	case roll < 55:
		playerTags.Apply(TagBleeding)
	case roll < 75:
		playerTags.Apply(TagLoud)
	default:
		playerTags.Apply(TagCursed)
	}
}

// PropagateKillTags applies tags when an enemy dies.
func PropagateKillTags(rng *rand.Rand, playerTags *TagSet, roomTags *TagSet, killCount int, projHadFire, projHadIce, projHadPoison bool) {
	// Element propagation
	if projHadFire {
		roomTags.Apply(TagScorched)
	}
	if projHadIce {
		roomTags.Apply(TagFrozen)
	}
	if projHadPoison {
		roomTags.Apply(TagDefiled)
	}

	// Killstreak → berserk
	if killCount >= 3 {
		playerTags.ApplyDuration(TagBerserk, 6)
	}

	// Blood accumulation
	roomTags.Apply(TagBloodSoaked)

	// Clean room tracking: remove quiet if combat got messy
	if roomTags.Has(TagContested) || roomTags.Has(TagDebris) {
		roomTags.Remove(TagQuiet)
	}
}

// EvaluateEnemyTags updates entity tags based on current state.
func EvaluateEnemyTags(e *Enemy, playerTags *TagSet, aliveCount, deadCount int) {
	// Cornered: low HP
	if e.HP > 0 && e.HP <= e.MaxHP/4 {
		if !e.Tags.Has(TagCornered) {
			e.Tags.Apply(TagCornered)
		}
	}

	// Frenzied: many allies dead
	if deadCount >= 2 && aliveCount <= 2 {
		e.Tags.Apply(TagFrenzied)
	}

	// Bloodhound: player is bleeding
	if playerTags.Has(TagBleeding) {
		e.Tags.ApplyDuration(TagBloodhound, 5)
	}

	// Aggressive: room is contested
	if e.Tags.Has(TagAlert) || e.Tags.Has(TagFrenzied) {
		e.Tags.Apply(TagAggressive)
	}
}
