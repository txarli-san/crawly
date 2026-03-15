package main

// Tag represents a modifier type that items can add to the player.
// Tags stack — picking up multiple items with the same tag increases the effect.
type Tag int

const (
	TagDamageUp  Tag = iota // +25% damage per stack
	TagSpeedUp              // +15% move speed per stack
	TagFireRateUp           // +20% fire rate per stack
	TagMaxHPUp              // +1 max HP per stack
	TagFire                 // burn DoT on enemies
	TagIce                  // slow enemies
	TagPoison               // poison DoT on enemies
	TagSplit                // +2 extra projectiles per stack
	TagPierce               // +1 pierce through enemies per stack
	TagBounce               // +2 wall bounces per stack
	TagHoming               // slight projectile tracking
	TagVampiric             // heal 1 HP on kill per stack
	TagExplosive            // AoE explosion on impact
	TagShield               // absorb 1 hit per stack (per room)
	TagChain                // damage chains to +1 nearby enemy per stack
	TagGiant                // +40% projectile size per stack
	TagCount
)

// ItemDef defines a collectible item and its tag contributions.
type ItemDef struct {
	Name   string
	Desc   string
	Tags   [TagCount]int
	Rarity int // 1=common, 2=uncommon, 3=rare
}

// Helper to build tag arrays concisely.
func itags(pairs ...int) [TagCount]int {
	var t [TagCount]int
	for i := 0; i+1 < len(pairs); i += 2 {
		t[Tag(pairs[i])] = pairs[i+1]
	}
	return t
}

var AllItemDefs = []ItemDef{
	// --- Common (Rarity 1) ---
	{Name: "Iron Tip", Desc: "+25% damage", Rarity: 1,
		Tags: itags(int(TagDamageUp), 1)},
	{Name: "Swift Boots", Desc: "+15% move speed", Rarity: 1,
		Tags: itags(int(TagSpeedUp), 1)},
	{Name: "Rapid Fire", Desc: "+20% fire rate", Rarity: 1,
		Tags: itags(int(TagFireRateUp), 1)},
	{Name: "Heart Container", Desc: "+2 max HP", Rarity: 1,
		Tags: itags(int(TagMaxHPUp), 2)},
	{Name: "Flame Shard", Desc: "Projectiles burn", Rarity: 1,
		Tags: itags(int(TagFire), 1)},
	{Name: "Frost Core", Desc: "Projectiles slow", Rarity: 1,
		Tags: itags(int(TagIce), 1)},
	{Name: "Venom Gland", Desc: "Projectiles poison", Rarity: 1,
		Tags: itags(int(TagPoison), 1)},
	{Name: "Rubber Ball", Desc: "+2 wall bounces", Rarity: 1,
		Tags: itags(int(TagBounce), 1)},

	// --- Uncommon (Rarity 2) ---
	{Name: "Prism Lens", Desc: "+2 extra shots", Rarity: 2,
		Tags: itags(int(TagSplit), 1)},
	{Name: "Ghost Arrow", Desc: "Pierce 1 enemy", Rarity: 2,
		Tags: itags(int(TagPierce), 1)},
	{Name: "Seeking Eye", Desc: "Homing projectiles", Rarity: 2,
		Tags: itags(int(TagHoming), 1)},
	{Name: "Blood Ruby", Desc: "Heal on kill", Rarity: 2,
		Tags: itags(int(TagVampiric), 1)},
	{Name: "Bomb Seed", Desc: "Explosive shots", Rarity: 2,
		Tags: itags(int(TagExplosive), 1)},
	{Name: "Barrier Gem", Desc: "Shield: absorb 1 hit", Rarity: 2,
		Tags: itags(int(TagShield), 1)},
	{Name: "Chain Link", Desc: "Damage chains to +1 enemy", Rarity: 2,
		Tags: itags(int(TagChain), 1)},
	{Name: "Growth Elixir", Desc: "+40% projectile size", Rarity: 2,
		Tags: itags(int(TagGiant), 1)},

	// --- Rare (Rarity 3) — multi-tag items ---
	{Name: "Demon Horn", Desc: "Damage + Fire", Rarity: 3,
		Tags: itags(int(TagDamageUp), 2, int(TagFire), 1)},
	{Name: "Frozen Heart", Desc: "HP + Ice", Rarity: 3,
		Tags: itags(int(TagMaxHPUp), 2, int(TagIce), 1)},
	{Name: "Scatter Shot", Desc: "Split + Pierce", Rarity: 3,
		Tags: itags(int(TagSplit), 1, int(TagPierce), 1)},
	{Name: "Bouncing Bomb", Desc: "Bounce + Explosive", Rarity: 3,
		Tags: itags(int(TagBounce), 1, int(TagExplosive), 1)},
	{Name: "Vampiric Burst", Desc: "Vampiric + Chain", Rarity: 3,
		Tags: itags(int(TagVampiric), 1, int(TagChain), 1)},
	{Name: "Inferno Lens", Desc: "Fire + Split", Rarity: 3,
		Tags: itags(int(TagFire), 1, int(TagSplit), 1)},
	{Name: "Frost Giant", Desc: "Ice + Giant", Rarity: 3,
		Tags: itags(int(TagIce), 1, int(TagGiant), 1)},
	{Name: "Toxic Missiles", Desc: "Poison + Homing", Rarity: 3,
		Tags: itags(int(TagPoison), 1, int(TagHoming), 1)},
}

// PlayerStats computed from accumulated item tags.
type PlayerStats struct {
	Damage    float32
	Speed     float32
	FireRate  float32 // shots per second
	MaxHP     int
	ProjCount int     // total projectiles per shot
	ProjSpeed float32 // projectile speed
	ProjSize  float32 // projectile radius

	// Tag stacks
	FireStacks    int
	IceStacks     int
	PoisonStacks  int
	PierceCount   int
	BounceCount   int
	HomingStacks  int
	VampiricStacks int
	ExplosiveStacks int
	ShieldStacks  int
	ChainStacks   int
	GiantStacks   int
}

// Base player stats
const (
	BaseDamage    = 3.0
	BaseSpeed     = 5.0
	BaseFireRate  = 3.0 // 3 shots/sec
	BaseHP        = 6
	BaseProjSpeed = 14.0
	BaseProjSize  = 0.08
)

// ComputeStats calculates effective stats from a list of collected items.
func ComputeStats(items []*ItemDef) PlayerStats {
	// Accumulate all tags
	var tags [TagCount]int
	for _, item := range items {
		for i := 0; i < int(TagCount); i++ {
			tags[i] += item.Tags[i]
		}
	}

	s := PlayerStats{
		Damage:    BaseDamage * (1.0 + 0.25*float32(tags[TagDamageUp])),
		Speed:     BaseSpeed * (1.0 + 0.15*float32(tags[TagSpeedUp])),
		FireRate:  BaseFireRate * (1.0 + 0.20*float32(tags[TagFireRateUp])),
		MaxHP:     BaseHP + tags[TagMaxHPUp],
		ProjCount: 1 + tags[TagSplit]*2,
		ProjSpeed: BaseProjSpeed,
		ProjSize:  BaseProjSize * (1.0 + 0.40*float32(tags[TagGiant])),

		FireStacks:      tags[TagFire],
		IceStacks:       tags[TagIce],
		PoisonStacks:    tags[TagPoison],
		PierceCount:     tags[TagPierce],
		BounceCount:     tags[TagBounce] * 2,
		HomingStacks:    tags[TagHoming],
		VampiricStacks:  tags[TagVampiric],
		ExplosiveStacks: tags[TagExplosive],
		ShieldStacks:    tags[TagShield],
		ChainStacks:     tags[TagChain],
		GiantStacks:     tags[TagGiant],
	}

	return s
}

// RollItemDrop picks a random item based on depth-scaled rarity weights.
func RollItemDrop(rng interface{ Float64() float64; Intn(int) int }, depth int) *ItemDef {
	// Higher depth → better rarity odds
	// Base weights: common=60, uncommon=30, rare=10
	// Per depth: common-2, uncommon+1, rare+1 (clamped)
	commonW := 60 - depth*2
	if commonW < 20 {
		commonW = 20
	}
	uncommonW := 30 + depth
	if uncommonW > 50 {
		uncommonW = 50
	}
	rareW := 10 + depth
	if rareW > 30 {
		rareW = 30
	}
	total := commonW + uncommonW + rareW
	roll := rng.Intn(total)

	var targetRarity int
	if roll < commonW {
		targetRarity = 1
	} else if roll < commonW+uncommonW {
		targetRarity = 2
	} else {
		targetRarity = 3
	}

	// Collect items of target rarity
	var candidates []*ItemDef
	for i := range AllItemDefs {
		if AllItemDefs[i].Rarity == targetRarity {
			candidates = append(candidates, &AllItemDefs[i])
		}
	}
	if len(candidates) == 0 {
		return &AllItemDefs[0] // fallback
	}
	return candidates[rng.Intn(len(candidates))]
}
