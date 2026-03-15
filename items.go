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

	// Melee tags (Warrior)
	TagCleave               // +30 degrees arc per stack
	TagLifesteal            // heal on melee hit
	TagThorns               // reflect damage when hit
	TagShockwave            // AoE burst on kill
	TagFury                 // faster swings per stack
	TagReach                // +0.5 melee range per stack

	TagCount
)

// ItemDef defines a collectible item and its tag contributions.
type ItemDef struct {
	Name   string
	Desc   string
	Tags   [TagCount]int
	Rarity int         // 1=common, 2=uncommon, 3=rare
	Class  PlayerClass // 0=mage (default), 1=warrior. -1 would mean shared but we use 0/1.
	Shared bool        // true = available to all classes
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
	// ===== SHARED (both classes) =====
	{Name: "Iron Tip", Desc: "+25% damage", Rarity: 1, Shared: true,
		Tags: itags(int(TagDamageUp), 1)},
	{Name: "Swift Boots", Desc: "+15% move speed", Rarity: 1, Shared: true,
		Tags: itags(int(TagSpeedUp), 1)},
	{Name: "Heart Container", Desc: "+2 max HP", Rarity: 1, Shared: true,
		Tags: itags(int(TagMaxHPUp), 2)},
	{Name: "Flame Shard", Desc: "Attacks burn", Rarity: 1, Shared: true,
		Tags: itags(int(TagFire), 1)},
	{Name: "Frost Core", Desc: "Attacks slow", Rarity: 1, Shared: true,
		Tags: itags(int(TagIce), 1)},
	{Name: "Venom Gland", Desc: "Attacks poison", Rarity: 1, Shared: true,
		Tags: itags(int(TagPoison), 1)},
	{Name: "Blood Ruby", Desc: "Heal on kill", Rarity: 2, Shared: true,
		Tags: itags(int(TagVampiric), 1)},
	{Name: "Barrier Gem", Desc: "Shield: absorb 1 hit", Rarity: 2, Shared: true,
		Tags: itags(int(TagShield), 1)},
	{Name: "Demon Horn", Desc: "Damage + Fire", Rarity: 3, Shared: true,
		Tags: itags(int(TagDamageUp), 2, int(TagFire), 1)},
	{Name: "Frozen Heart", Desc: "HP + Ice", Rarity: 3, Shared: true,
		Tags: itags(int(TagMaxHPUp), 2, int(TagIce), 1)},

	// ===== MAGE =====
	{Name: "Rapid Fire", Desc: "+20% fire rate", Rarity: 1, Class: ClassMage,
		Tags: itags(int(TagFireRateUp), 1)},
	{Name: "Rubber Ball", Desc: "+2 wall bounces", Rarity: 1, Class: ClassMage,
		Tags: itags(int(TagBounce), 1)},
	{Name: "Prism Lens", Desc: "+2 extra shots", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagSplit), 1)},
	{Name: "Ghost Arrow", Desc: "Pierce 1 enemy", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagPierce), 1)},
	{Name: "Seeking Eye", Desc: "Homing projectiles", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagHoming), 1)},
	{Name: "Bomb Seed", Desc: "Explosive shots", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagExplosive), 1)},
	{Name: "Chain Link", Desc: "Chains to +1 enemy", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagChain), 1)},
	{Name: "Growth Elixir", Desc: "+40% projectile size", Rarity: 2, Class: ClassMage,
		Tags: itags(int(TagGiant), 1)},
	{Name: "Scatter Shot", Desc: "Split + Pierce", Rarity: 3, Class: ClassMage,
		Tags: itags(int(TagSplit), 1, int(TagPierce), 1)},
	{Name: "Bouncing Bomb", Desc: "Bounce + Explosive", Rarity: 3, Class: ClassMage,
		Tags: itags(int(TagBounce), 1, int(TagExplosive), 1)},
	{Name: "Inferno Lens", Desc: "Fire + Split", Rarity: 3, Class: ClassMage,
		Tags: itags(int(TagFire), 1, int(TagSplit), 1)},
	{Name: "Toxic Missiles", Desc: "Poison + Homing", Rarity: 3, Class: ClassMage,
		Tags: itags(int(TagPoison), 1, int(TagHoming), 1)},

	// ===== WARRIOR =====
	{Name: "Whetstone", Desc: "+30° swing arc", Rarity: 1, Class: ClassWarrior,
		Tags: itags(int(TagCleave), 1)},
	{Name: "Battle Fury", Desc: "Faster swings", Rarity: 1, Class: ClassWarrior,
		Tags: itags(int(TagFury), 1)},
	{Name: "Long Blade", Desc: "+0.5 reach", Rarity: 1, Class: ClassWarrior,
		Tags: itags(int(TagReach), 1)},
	{Name: "Leech Fang", Desc: "Heal on melee hit", Rarity: 2, Class: ClassWarrior,
		Tags: itags(int(TagLifesteal), 1)},
	{Name: "Spiked Armor", Desc: "Reflect damage", Rarity: 2, Class: ClassWarrior,
		Tags: itags(int(TagThorns), 1)},
	{Name: "War Drum", Desc: "AoE burst on kill", Rarity: 2, Class: ClassWarrior,
		Tags: itags(int(TagShockwave), 1)},
	{Name: "Berserker Axe", Desc: "Damage + Fury", Rarity: 3, Class: ClassWarrior,
		Tags: itags(int(TagDamageUp), 2, int(TagFury), 1)},
	{Name: "Thorn Mail", Desc: "Thorns + HP", Rarity: 3, Class: ClassWarrior,
		Tags: itags(int(TagThorns), 1, int(TagMaxHPUp), 3)},
	{Name: "Cleaving Flame", Desc: "Cleave + Fire", Rarity: 3, Class: ClassWarrior,
		Tags: itags(int(TagCleave), 1, int(TagFire), 1)},
	{Name: "Vampiric Blade", Desc: "Lifesteal + Reach", Rarity: 3, Class: ClassWarrior,
		Tags: itags(int(TagLifesteal), 1, int(TagReach), 1)},
}

// PlayerStats computed from class base + accumulated item tags.
type PlayerStats struct {
	Damage    float32
	Speed     float32
	FireRate  float32 // shots per second (Mage)
	MaxHP     int
	ProjCount int     // projectiles per shot (Mage)
	ProjSpeed float32
	ProjSize  float32

	// Melee (Warrior/Rogue)
	MeleeArc      float32 // degrees
	MeleeRange    float32 // in tile units
	MeleeCooldown float32 // seconds
	NoiseRadius   float32 // tiles

	// Tag stacks
	FireStacks      int
	IceStacks       int
	PoisonStacks    int
	PierceCount     int
	BounceCount     int
	HomingStacks    int
	VampiricStacks  int
	ExplosiveStacks int
	ShieldStacks    int
	ChainStacks     int
	GiantStacks     int

	// Melee stacks
	CleaveStacks    int
	LifestealStacks int
	ThornsStacks    int
	ShockwaveStacks int
	FuryStacks      int
	ReachStacks     int
}

// ComputeStats calculates effective stats from class + collected items.
func ComputeStats(class PlayerClass, items []*ItemDef) PlayerStats {
	// Class base stats
	var baseDmg, baseSpd, baseFireRate, baseProjSpd, baseProjSize float32
	var baseHP int
	var meleeArc, meleeRange, meleeCooldown, noiseRadius float32

	switch class {
	case ClassMage:
		baseDmg, baseSpd, baseFireRate = 3.0, 5.0, 3.0
		baseHP = 6
		baseProjSpd, baseProjSize = 14.0, 0.08
		noiseRadius = 6.0
	case ClassWarrior:
		baseDmg, baseSpd = 5.0, 4.0
		baseHP = 10
		meleeArc, meleeRange, meleeCooldown = 120, 1.8, 0.5
		noiseRadius = 8.0
	}

	// Accumulate item tags
	var tags [TagCount]int
	for _, item := range items {
		for i := 0; i < int(TagCount); i++ {
			tags[i] += item.Tags[i]
		}
	}

	s := PlayerStats{
		Damage:        baseDmg * (1.0 + 0.25*float32(tags[TagDamageUp])),
		Speed:         baseSpd * (1.0 + 0.15*float32(tags[TagSpeedUp])),
		FireRate:      baseFireRate * (1.0 + 0.20*float32(tags[TagFireRateUp])),
		MaxHP:         baseHP + tags[TagMaxHPUp],
		ProjCount:     1 + tags[TagSplit]*2,
		ProjSpeed:     baseProjSpd,
		ProjSize:      baseProjSize * (1.0 + 0.40*float32(tags[TagGiant])),
		MeleeArc:      meleeArc,
		MeleeRange:    meleeRange,
		MeleeCooldown: meleeCooldown,
		NoiseRadius:   noiseRadius,

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

		CleaveStacks:    tags[TagCleave],
		LifestealStacks: tags[TagLifesteal],
		ThornsStacks:    tags[TagThorns],
		ShockwaveStacks: tags[TagShockwave],
		FuryStacks:      tags[TagFury],
		ReachStacks:     tags[TagReach],
	}

	// Apply melee tags to stats
	s.MeleeArc += float32(s.CleaveStacks) * 30
	s.MeleeRange += float32(s.ReachStacks) * 0.5
	if s.FuryStacks > 0 {
		s.MeleeCooldown *= 1.0 / (1.0 + 0.2*float32(s.FuryStacks))
	}

	return s
}

// RollItemDrop picks a random item based on depth-scaled rarity weights.
func RollItemDrop(rng interface{ Float64() float64; Intn(int) int }, depth int, class PlayerClass) *ItemDef {
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

	// Collect items of target rarity matching class
	var candidates []*ItemDef
	for i := range AllItemDefs {
		item := &AllItemDefs[i]
		if item.Rarity != targetRarity {
			continue
		}
		if item.Shared || item.Class == class {
			candidates = append(candidates, item)
		}
	}
	if len(candidates) == 0 {
		return &AllItemDefs[0] // fallback
	}
	return candidates[rng.Intn(len(candidates))]
}
