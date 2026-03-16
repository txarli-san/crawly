package main

import (
	"math/rand"
	"testing"
)

// ---------- itags helper ----------

func TestItagsEmpty(t *testing.T) {
	tags := itags()
	for i := 0; i < int(TagCount); i++ {
		if tags[i] != 0 {
			t.Fatalf("expected zero tags, got %d at index %d", tags[i], i)
		}
	}
}

func TestItagsSingle(t *testing.T) {
	tags := itags(int(TagFire), 3)
	if tags[TagFire] != 3 {
		t.Fatalf("expected TagFire=3, got %d", tags[TagFire])
	}
	// everything else zero
	for i := 0; i < int(TagCount); i++ {
		if Tag(i) != TagFire && tags[i] != 0 {
			t.Fatalf("unexpected non-zero tag at index %d: %d", i, tags[i])
		}
	}
}

func TestItagsMultiple(t *testing.T) {
	tags := itags(int(TagDamageUp), 2, int(TagIce), 1)
	if tags[TagDamageUp] != 2 {
		t.Fatalf("expected TagDamageUp=2, got %d", tags[TagDamageUp])
	}
	if tags[TagIce] != 1 {
		t.Fatalf("expected TagIce=1, got %d", tags[TagIce])
	}
}

func TestItagsOddArgCount(t *testing.T) {
	// Odd number of args — last one has no value, should not panic
	tags := itags(int(TagFire), 1, int(TagPoison))
	if tags[TagFire] != 1 {
		t.Fatalf("expected TagFire=1, got %d", tags[TagFire])
	}
	// TagPoison should stay 0 since it has no pair
	if tags[TagPoison] != 0 {
		t.Fatalf("expected TagPoison=0 (no pair), got %d", tags[TagPoison])
	}
}

// ---------- ComputeStats ----------

func TestComputeStatsMageBaseNoItems(t *testing.T) {
	s := ComputeStats(ClassMage, nil)
	if s.Damage != 3.0 {
		t.Errorf("mage base damage: want 3.0, got %f", s.Damage)
	}
	if s.Speed != 5.0 {
		t.Errorf("mage base speed: want 5.0, got %f", s.Speed)
	}
	if s.FireRate != 3.0 {
		t.Errorf("mage base fire rate: want 3.0, got %f", s.FireRate)
	}
	if s.MaxHP != 6 {
		t.Errorf("mage base HP: want 6, got %d", s.MaxHP)
	}
	if s.ProjCount != 1 {
		t.Errorf("mage base proj count: want 1, got %d", s.ProjCount)
	}
	if s.ProjSpeed != 14.0 {
		t.Errorf("mage base proj speed: want 14.0, got %f", s.ProjSpeed)
	}
	// no melee stats
	if s.MeleeArc != 0 || s.MeleeRange != 0 || s.MeleeCooldown != 0 {
		t.Errorf("mage should have zero melee stats")
	}
}

func TestComputeStatsWarriorBaseNoItems(t *testing.T) {
	s := ComputeStats(ClassWarrior, nil)
	if s.Damage != 5.0 {
		t.Errorf("warrior base damage: want 5.0, got %f", s.Damage)
	}
	if s.Speed != 4.0 {
		t.Errorf("warrior base speed: want 4.0, got %f", s.Speed)
	}
	if s.MaxHP != 10 {
		t.Errorf("warrior base HP: want 10, got %d", s.MaxHP)
	}
	if s.MeleeArc != 120 {
		t.Errorf("warrior base melee arc: want 120, got %f", s.MeleeArc)
	}
	if s.MeleeRange != 1.8 {
		t.Errorf("warrior base melee range: want 1.8, got %f", s.MeleeRange)
	}
	if s.MeleeCooldown != 0.5 {
		t.Errorf("warrior base melee cooldown: want 0.5, got %f", s.MeleeCooldown)
	}
	// no ranged stats
	if s.FireRate != 0 || s.ProjSpeed != 0 {
		t.Errorf("warrior should have zero ranged stats")
	}
}

func TestComputeStatsDamageStacking(t *testing.T) {
	ironTip := findItem("Iron Tip")
	// 1 stack: 3.0 * (1 + 0.25) = 3.75
	s := ComputeStats(ClassMage, []*ItemDef{ironTip})
	want := float32(3.0 * 1.25)
	if s.Damage != want {
		t.Errorf("1 stack damage: want %f, got %f", want, s.Damage)
	}
	// 3 stacks: 3.0 * (1 + 0.75) = 5.25
	s = ComputeStats(ClassMage, []*ItemDef{ironTip, ironTip, ironTip})
	want = float32(3.0 * 1.75)
	if s.Damage != want {
		t.Errorf("3 stack damage: want %f, got %f", want, s.Damage)
	}
}

func TestComputeStatsSpeedStacking(t *testing.T) {
	boots := findItem("Swift Boots")
	// 2 stacks: 5.0 * (1 + 0.30) = 6.5
	s := ComputeStats(ClassMage, []*ItemDef{boots, boots})
	want := float32(5.0 * 1.30)
	if s.Speed != want {
		t.Errorf("2 stack speed: want %f, got %f", want, s.Speed)
	}
}

func TestComputeStatsMaxHPAdditive(t *testing.T) {
	heart := findItem("Heart Container") // +2 HP
	frozenH := findItem("Frozen Heart")  // +2 HP + ice
	s := ComputeStats(ClassMage, []*ItemDef{heart, frozenH})
	// 6 base + 2 + 2 = 10
	if s.MaxHP != 10 {
		t.Errorf("max HP: want 10, got %d", s.MaxHP)
	}
}

func TestComputeStatsSplitProjectiles(t *testing.T) {
	prism := findItem("Prism Lens")
	// 1 stack: 1 + 1*2 = 3 projectiles
	s := ComputeStats(ClassMage, []*ItemDef{prism})
	if s.ProjCount != 3 {
		t.Errorf("1 split proj count: want 3, got %d", s.ProjCount)
	}
	// 2 stacks: 1 + 2*2 = 5
	s = ComputeStats(ClassMage, []*ItemDef{prism, prism})
	if s.ProjCount != 5 {
		t.Errorf("2 split proj count: want 5, got %d", s.ProjCount)
	}
}

func TestComputeStatsBounceMultiplier(t *testing.T) {
	rubber := findItem("Rubber Ball")
	// 1 bounce tag stack → BounceCount = 1*2 = 2
	s := ComputeStats(ClassMage, []*ItemDef{rubber})
	if s.BounceCount != 2 {
		t.Errorf("bounce count: want 2, got %d", s.BounceCount)
	}
}

func TestComputeStatsGiantProjectileSize(t *testing.T) {
	growth := findItem("Growth Elixir")
	s := ComputeStats(ClassMage, []*ItemDef{growth, growth})
	// 0.08 * (1 + 0.40*2) = 0.08 * 1.80 = 0.144
	want := float32(0.08 * 1.80)
	diff := s.ProjSize - want
	if diff > 0.001 || diff < -0.001 {
		t.Errorf("proj size: want ~%f, got %f", want, s.ProjSize)
	}
}

func TestComputeStatsWarriorCleave(t *testing.T) {
	whet := findItem("Whetstone")
	s := ComputeStats(ClassWarrior, []*ItemDef{whet, whet})
	// 120 base + 2*30 = 180
	if s.MeleeArc != 180 {
		t.Errorf("cleave arc: want 180, got %f", s.MeleeArc)
	}
}

func TestComputeStatsWarriorReach(t *testing.T) {
	blade := findItem("Long Blade")
	s := ComputeStats(ClassWarrior, []*ItemDef{blade, blade, blade})
	// 1.8 + 3*0.5 = 3.3
	want := float32(1.8 + 1.5)
	diff := s.MeleeRange - want
	if diff > 0.001 || diff < -0.001 {
		t.Errorf("melee range: want %f, got %f", want, s.MeleeRange)
	}
}

func TestComputeStatsWarriorFuryCooldown(t *testing.T) {
	fury := findItem("Battle Fury")
	s := ComputeStats(ClassWarrior, []*ItemDef{fury})
	// 0.5 * 1/(1+0.2) = 0.5/1.2 ≈ 0.4167
	want := float32(0.5 / 1.2)
	diff := s.MeleeCooldown - want
	if diff > 0.001 || diff < -0.001 {
		t.Errorf("fury cooldown: want ~%f, got %f", want, s.MeleeCooldown)
	}
	// 3 stacks: 0.5 / (1+0.6) = 0.3125
	s = ComputeStats(ClassWarrior, []*ItemDef{fury, fury, fury})
	want = float32(0.5 / 1.6)
	diff = s.MeleeCooldown - want
	if diff > 0.001 || diff < -0.001 {
		t.Errorf("3x fury cooldown: want ~%f, got %f", want, s.MeleeCooldown)
	}
}

func TestComputeStatsMultiTagItem(t *testing.T) {
	demon := findItem("Demon Horn") // DamageUp=2, Fire=1
	s := ComputeStats(ClassMage, []*ItemDef{demon})
	// damage: 3.0 * (1 + 0.25*2) = 4.5
	if s.Damage != 4.5 {
		t.Errorf("demon horn damage: want 4.5, got %f", s.Damage)
	}
	if s.FireStacks != 1 {
		t.Errorf("demon horn fire stacks: want 1, got %d", s.FireStacks)
	}
}

func TestComputeStatsWarriorNoFuryNoCooldownChange(t *testing.T) {
	// No fury items — cooldown should stay at base
	s := ComputeStats(ClassWarrior, nil)
	if s.MeleeCooldown != 0.5 {
		t.Errorf("no fury cooldown: want 0.5, got %f", s.MeleeCooldown)
	}
}

func TestComputeStatsEmptyItemSlice(t *testing.T) {
	s := ComputeStats(ClassMage, []*ItemDef{})
	base := ComputeStats(ClassMage, nil)
	if s != base {
		t.Errorf("empty slice stats should equal nil stats")
	}
}

func TestComputeStatsMageWithWarriorItems(t *testing.T) {
	// Mage picks up warrior items — tags still accumulate,
	// but melee stats stay zero (no base melee)
	whet := findItem("Whetstone")
	s := ComputeStats(ClassMage, []*ItemDef{whet})
	if s.CleaveStacks != 1 {
		t.Errorf("mage with whetstone should have CleaveStacks=1, got %d", s.CleaveStacks)
	}
	// Melee arc is 0 base + 30 = 30, but no range/cooldown
	if s.MeleeArc != 30 {
		t.Errorf("mage melee arc: want 30, got %f", s.MeleeArc)
	}
	if s.MeleeRange != 0 {
		t.Errorf("mage melee range should be 0, got %f", s.MeleeRange)
	}
}

// ---------- RollItemDrop ----------

func TestRollItemDropAlwaysReturnsItem(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		item := RollItemDrop(rng, i%30, ClassMage)
		if item == nil {
			t.Fatalf("RollItemDrop returned nil at iteration %d", i)
		}
		if item.Name == "" {
			t.Fatalf("RollItemDrop returned item with empty name at iteration %d", i)
		}
	}
}

func TestRollItemDropClassFiltering(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for i := 0; i < 500; i++ {
		item := RollItemDrop(rng, 5, ClassMage)
		if !item.Shared && item.Class != ClassMage {
			t.Fatalf("mage got warrior item: %s", item.Name)
		}
	}
	for i := 0; i < 500; i++ {
		item := RollItemDrop(rng, 5, ClassWarrior)
		if !item.Shared && item.Class != ClassWarrior {
			t.Fatalf("warrior got mage item: %s", item.Name)
		}
	}
}

func TestRollItemDropRarityShiftsWithDepth(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	runs := 5000

	countRarity := func(depth int) [4]int {
		var counts [4]int
		for i := 0; i < runs; i++ {
			item := RollItemDrop(rng, depth, ClassMage)
			counts[item.Rarity]++
		}
		return counts
	}

	shallow := countRarity(0)
	deep := countRarity(20)

	// At depth 0: common should dominate
	if shallow[1] < shallow[3] {
		t.Errorf("at depth 0, common (%d) should exceed rare (%d)", shallow[1], shallow[3])
	}
	// At depth 20: rare should be more frequent than at depth 0
	if deep[3] <= shallow[3] {
		t.Errorf("rare at depth 20 (%d) should exceed rare at depth 0 (%d)", deep[3], shallow[3])
	}
}

func TestRollItemDropRarityWeightClamping(t *testing.T) {
	// At extreme depth, weights should clamp
	rng := rand.New(rand.NewSource(1))
	// depth=100: commonW=max(60-200,20)=20, uncommonW=min(130,50)=50, rareW=min(110,30)=30
	// total=100, so distribution is 20/50/30
	var counts [4]int
	runs := 10000
	for i := 0; i < runs; i++ {
		item := RollItemDrop(rng, 100, ClassMage)
		counts[item.Rarity]++
	}
	// Uncommon should be highest
	if counts[2] < counts[1] || counts[2] < counts[3] {
		t.Errorf("at extreme depth, uncommon (%d) should be most frequent (common=%d rare=%d)",
			counts[2], counts[1], counts[3])
	}
}

func TestRollItemDropDepthZero(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	// depth 0 weights: common=60, uncommon=30, rare=10
	var counts [4]int
	runs := 10000
	for i := 0; i < runs; i++ {
		item := RollItemDrop(rng, 0, ClassWarrior)
		counts[item.Rarity]++
	}
	// Common should be at least 50% of drops
	if counts[1] < runs/2 {
		t.Errorf("at depth 0, expected >50%% common, got %d/%d", counts[1], runs)
	}
}

// ---------- AllItemDefs integrity ----------

func TestAllItemDefsNoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, item := range AllItemDefs {
		if seen[item.Name] {
			t.Errorf("duplicate item name: %s", item.Name)
		}
		seen[item.Name] = true
	}
}

func TestAllItemDefsHaveValidRarity(t *testing.T) {
	for _, item := range AllItemDefs {
		if item.Rarity < 1 || item.Rarity > 3 {
			t.Errorf("item %q has invalid rarity %d", item.Name, item.Rarity)
		}
	}
}

func TestAllItemDefsHaveAtLeastOneTag(t *testing.T) {
	for _, item := range AllItemDefs {
		hasTag := false
		for i := 0; i < int(TagCount); i++ {
			if item.Tags[i] != 0 {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("item %q has no tags", item.Name)
		}
	}
}

func TestAllItemDefsHaveDescription(t *testing.T) {
	for _, item := range AllItemDefs {
		if item.Desc == "" {
			t.Errorf("item %q has empty description", item.Name)
		}
	}
}

func TestAllItemDefsClassOrShared(t *testing.T) {
	for _, item := range AllItemDefs {
		if !item.Shared && item.Class != ClassMage && item.Class != ClassWarrior {
			t.Errorf("item %q is not shared and has unknown class %d", item.Name, item.Class)
		}
	}
}

func TestAllItemDefsNoNegativeTags(t *testing.T) {
	for _, item := range AllItemDefs {
		for i := 0; i < int(TagCount); i++ {
			if item.Tags[i] < 0 {
				t.Errorf("item %q has negative tag value at index %d: %d", item.Name, i, item.Tags[i])
			}
		}
	}
}

func TestEveryRarityHasItemsForBothClasses(t *testing.T) {
	for rarity := 1; rarity <= 3; rarity++ {
		for _, class := range []PlayerClass{ClassMage, ClassWarrior} {
			found := false
			for _, item := range AllItemDefs {
				if item.Rarity == rarity && (item.Shared || item.Class == class) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("no rarity %d items for class %d", rarity, class)
			}
		}
	}
}

// ---------- helpers ----------

func findItem(name string) *ItemDef {
	for i := range AllItemDefs {
		if AllItemDefs[i].Name == name {
			return &AllItemDefs[i]
		}
	}
	panic("item not found: " + name)
}
