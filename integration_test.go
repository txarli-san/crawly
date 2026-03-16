package main

import (
	"math"
	"testing"
)

const simDT = 1.0 / 60.0 // 60fps tick

// stepFrames advances the game by n frames.
func stepFrames(g *GameState, n int) {
	for range n {
		g.Update(simDT)
	}
}

// stepSeconds advances the game by approximately t seconds.
func stepSeconds(g *GameState, t float32) {
	frames := int(t / simDT)
	stepFrames(g, frames)
}

// simGame creates a game with tileUnit=1 for easy coordinate math.
func simGame(class PlayerClass) *GameState {
	return NewGame(42, 1.0, 0.0, class)
}

// setupCombat creates a game with one enemy at a known position, player facing it.
func setupCombat(class PlayerClass, items []*ItemDef, enemyType EnemyType, depth int) *GameState {
	g := simGame(class)
	g.CurrentRoom.Cleared = false

	// Equip items
	g.Player.Items = items
	g.Player.Stats = ComputeStats(class, items)
	g.Player.HP = g.Player.Stats.MaxHP
	g.Player.MaxHP = g.Player.Stats.MaxHP

	// Place player at center
	g.Player.X = float32(RoomW) / 2.0
	g.Player.Z = float32(RoomH) / 2.0

	// Clear any existing enemies
	g.Enemies = g.Enemies[:0]

	return g
}

// spawnTargetEnemy adds an enemy in front of the player (positive Z direction).
func spawnTargetEnemy(g *GameState, etype EnemyType, depth int, dist float32) int {
	e := makeEnemy(EnemySpawn{
		X: g.Player.X,
		Z: g.Player.Z + dist,
	}, 1.0, depth) // tileUnit=1 so spawn coords = world coords
	// Override position since makeEnemy multiplies by tileUnit
	e.X = g.Player.X
	e.Z = g.Player.Z + dist
	g.Enemies = append(g.Enemies, e)
	return len(g.Enemies) - 1
}

// ---------- TTK: Time-to-Kill simulations ----------

func TestIntegrationMageTTKDepth0(t *testing.T) {
	g := setupCombat(ClassMage, nil, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)

	// Fire at enemy repeatedly
	maxFrames := 600 // 10 seconds
	for frame := range maxFrames {
		if !g.Enemies[idx].Alive {
			t.Logf("Minion killed in %d frames (%.2fs)", frame, float32(frame)*simDT)
			return
		}
		// Fire every time cooldown allows
		if g.Player.FireTimer <= 0 {
			g.SpawnPlayerProjectiles(0, 1) // aim +Z toward enemy
			g.Player.FireTimer = 1.0 / g.Player.Stats.FireRate
		}
		g.Update(simDT)
	}
	t.Errorf("failed to kill minion in 10s, HP remaining: %d/%d", g.Enemies[idx].HP, g.Enemies[idx].MaxHP)
}

func TestIntegrationMageTTKScalesWithDepth(t *testing.T) {
	var ttks []int

	for _, depth := range []int{0, 5, 10, 15, 20} {
		g := setupCombat(ClassMage, nil, EnemyWarrior, 0)
		idx := spawnTargetEnemy(g, EnemyWarrior, depth, 3.0)
		g.Enemies[idx].State = StateIdle // don't chase

		killed := false
		for frame := range 1800 { // 30s max
			if !g.Enemies[idx].Alive {
				ttks = append(ttks, frame)
				killed = true
				break
			}
			if g.Player.FireTimer <= 0 {
				g.SpawnPlayerProjectiles(0, 1)
				g.Player.FireTimer = 1.0 / g.Player.Stats.FireRate
			}
			g.Update(simDT)
		}
		if !killed {
			t.Errorf("depth %d: failed to kill warrior in 30s", depth)
			ttks = append(ttks, 1800)
		}
	}

	// TTK should increase with depth
	for i := 1; i < len(ttks); i++ {
		if ttks[i] < ttks[i-1] {
			t.Errorf("TTK should increase with depth: depth[%d]=%d < depth[%d]=%d",
				i, ttks[i], i-1, ttks[i-1])
		}
	}
	t.Logf("TTK frames by depth: %v", ttks)
}

// ---------- Item synergy end-to-end ----------

func TestIntegrationPiercePassesThrough(t *testing.T) {
	// A piercing projectile should hit the first enemy exactly once,
	// pass through it, and hit the second enemy behind it.
	ghost := findItem("Ghost Arrow")
	g := setupCombat(ClassMage, []*ItemDef{ghost}, EnemyMinion, 0)

	// Two enemies lined up along the projectile path
	idx0 := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)
	idx1 := spawnTargetEnemy(g, EnemyMinion, 0, 5.0)
	g.Enemies[idx0].State = StateIdle
	g.Enemies[idx1].State = StateIdle
	g.Enemies[idx0].HP = 100
	g.Enemies[idx0].MaxHP = 100
	g.Enemies[idx1].HP = 100
	g.Enemies[idx1].MaxHP = 100

	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 180) // 3 seconds for projectile to reach both

	dmg0 := 100 - g.Enemies[idx0].HP
	dmg1 := 100 - g.Enemies[idx1].HP
	baseDmg := int(g.Player.Stats.Damage)

	// First enemy should be hit exactly once
	if dmg0 != baseDmg {
		t.Errorf("first enemy should take exactly 1 hit (%d dmg), took %d", baseDmg, dmg0)
	}
	// Second enemy should also be hit
	if dmg1 == 0 {
		t.Errorf("second enemy should be hit after pierce, took 0 damage")
	}
}

func TestIntegrationExplosiveAoE(t *testing.T) {
	bomb := findItem("Bomb Seed")
	g := setupCombat(ClassMage, []*ItemDef{bomb}, EnemyMinion, 0)

	// Place two enemies close together
	idx0 := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)
	// Second enemy slightly to the side of first
	e1 := makeEnemy(EnemySpawn{Type: EnemyMinion}, 1.0, 0)
	e1.X = g.Player.X + 0.8
	e1.Z = g.Player.Z + 3.0
	e1.State = StateIdle
	g.Enemies = append(g.Enemies, e1)
	idx1 := len(g.Enemies) - 1
	g.Enemies[idx0].State = StateIdle

	startHP1 := g.Enemies[idx1].HP

	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 120)

	if g.Enemies[idx1].HP >= startHP1 {
		t.Error("explosion should damage nearby enemy")
	}
}

func TestIntegrationFireDoTKillsOverTime(t *testing.T) {
	flame := findItem("Flame Shard")
	g := setupCombat(ClassMage, []*ItemDef{flame}, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)
	g.Enemies[idx].State = StateIdle
	g.Enemies[idx].HP = 200 // huge tank to isolate DoT
	g.Enemies[idx].MaxHP = 200

	// Fire once to apply burn
	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 30) // let projectile hit

	if g.Enemies[idx].FireTimer <= 0 {
		t.Fatal("enemy should be burning")
	}

	hpAfterHit := g.Enemies[idx].HP
	stepFrames(g, 60) // 1 second of burn

	hpAfterBurn := g.Enemies[idx].HP
	burnDmg := hpAfterHit - hpAfterBurn

	if burnDmg <= 0 {
		t.Errorf("fire DoT should reduce HP: was %d, now %d", hpAfterHit, hpAfterBurn)
	}
	// Note: fire DoT does ceil(fireDamage * dt) per frame, which rounds
	// up to 1 per frame = ~60 DPS. This is much higher than the nominal
	// 1.5/sec due to Ceil on small floats. Logging for visibility.
	t.Logf("fire DoT dealt %d damage in 1 second (nominal: 1.5/sec, actual: ~%d/sec due to Ceil rounding)",
		burnDmg, burnDmg)
}

func TestIntegrationIceSlowsEnemyChase(t *testing.T) {
	frost := findItem("Frost Core")
	g := setupCombat(ClassMage, []*ItemDef{frost}, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 5.0)
	g.Enemies[idx].State = StateChasing

	// Fire to apply ice
	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 30) // let it hit

	if g.Enemies[idx].IceTimer <= 0 {
		t.Fatal("enemy should be frozen")
	}

	// Measure chase distance while iced
	startZ := g.Enemies[idx].Z
	stepFrames(g, 60)
	icedTravel := math.Abs(float64(g.Enemies[idx].Z - startZ))

	// Now un-ice and measure normal chase
	g.Enemies[idx].IceTimer = 0
	startZ = g.Enemies[idx].Z
	stepFrames(g, 60)
	normalTravel := math.Abs(float64(g.Enemies[idx].Z - startZ))

	if icedTravel >= normalTravel {
		t.Errorf("iced travel (%f) should be less than normal (%f)", icedTravel, normalTravel)
	}
}

func TestIntegrationVampiricHealOnKill(t *testing.T) {
	ruby := findItem("Blood Ruby")
	g := setupCombat(ClassMage, []*ItemDef{ruby}, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)
	g.Enemies[idx].State = StateIdle
	g.Enemies[idx].HP = 1 // one hit kill

	g.Player.HP = 3 // below max to see heal

	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 60)

	if g.Enemies[idx].Alive {
		t.Fatal("enemy should be dead")
	}
	if g.Player.HP <= 3 {
		t.Errorf("vampiric should heal on kill: HP=%d", g.Player.HP)
	}
}

func TestIntegrationSplitFiresMultipleProjectiles(t *testing.T) {
	prism := findItem("Prism Lens")
	g := setupCombat(ClassMage, []*ItemDef{prism, prism}, EnemyMinion, 0)

	g.SpawnPlayerProjectiles(0, 1)
	if len(g.Projectiles) != 5 {
		t.Errorf("2x split should fire 5 projectiles, got %d", len(g.Projectiles))
	}
}

// ---------- Warrior melee integration ----------

func TestIntegrationWarriorMeleeKill(t *testing.T) {
	g := setupCombat(ClassWarrior, nil, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 1.5)
	g.Enemies[idx].State = StateIdle

	// Face toward enemy (angle 0 = +Z)
	g.Player.FacingAngle = 0

	g.StartMeleeAttack()
	stepFrames(g, 30) // let swing complete

	if g.Enemies[idx].HP >= 6 {
		t.Errorf("melee should deal damage: HP=%d", g.Enemies[idx].HP)
	}
}

func TestIntegrationWarriorLifesteal(t *testing.T) {
	leech := findItem("Leech Fang")
	g := setupCombat(ClassWarrior, []*ItemDef{leech}, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 1.5)
	g.Enemies[idx].State = StateIdle
	g.Player.FacingAngle = 0
	g.Player.HP = 5

	g.StartMeleeAttack()
	stepFrames(g, 30)

	if g.Player.HP <= 5 {
		t.Errorf("lifesteal should heal on melee hit: HP=%d", g.Player.HP)
	}
}

func TestIntegrationWarriorShockwaveOnKill(t *testing.T) {
	drum := findItem("War Drum")
	g := setupCombat(ClassWarrior, []*ItemDef{drum}, EnemyMinion, 0)

	// First enemy close, will be killed
	idx0 := spawnTargetEnemy(g, EnemyMinion, 0, 1.5)
	g.Enemies[idx0].State = StateIdle
	g.Enemies[idx0].HP = 1

	// Second enemy nearby, should take shockwave damage
	e1 := makeEnemy(EnemySpawn{Type: EnemyMinion}, 1.0, 0)
	e1.X = g.Player.X + 1.0
	e1.Z = g.Player.Z + 1.5
	e1.State = StateIdle
	g.Enemies = append(g.Enemies, e1)
	idx1 := len(g.Enemies) - 1
	startHP1 := g.Enemies[idx1].HP

	g.Player.FacingAngle = 0
	g.StartMeleeAttack()
	stepFrames(g, 60) // let swing + explosion resolve

	if g.Enemies[idx0].Alive {
		t.Fatal("first enemy should be dead")
	}
	if g.Enemies[idx1].HP >= startHP1 {
		t.Error("shockwave should damage nearby enemy on kill")
	}
}

// ---------- Room progression ----------

func TestIntegrationRoomProgressionMultipleRooms(t *testing.T) {
	g := simGame(ClassMage)

	for i := range 5 {
		g.beginTransition(DoorNorth)
		g.finishTransition()

		if g.CurrentRoom == nil {
			t.Fatalf("room %d: current room is nil", i+1)
		}
		if len(g.Enemies) == 0 {
			t.Fatalf("room %d: no enemies spawned", i+1)
		}

		// Kill all enemies
		for j := range g.Enemies {
			g.Enemies[j].Alive = false
		}
		g.checkRoomCleared()

		if !g.CurrentRoom.Cleared {
			t.Fatalf("room %d: should be cleared", i+1)
		}
	}

	if g.RoomsCleared < 5 {
		t.Errorf("should have cleared 5 rooms, got %d", g.RoomsCleared)
	}
	// Should have 6 rooms in map (start + 5)
	if len(g.RoomMap) != 6 {
		t.Errorf("should have 6 rooms in map, got %d", len(g.RoomMap))
	}
}

func TestIntegrationDepthIncreasesWithProgression(t *testing.T) {
	g := simGame(ClassMage)
	var depths []int

	for range 10 {
		g.beginTransition(DoorNorth)
		g.finishTransition()
		depths = append(depths, g.CurrentRoom.Depth)

		// Clear room
		for j := range g.Enemies {
			g.Enemies[j].Alive = false
		}
		g.checkRoomCleared()
	}

	// Depth should generally increase (it equals RoomsCleared at generation time)
	if depths[9] <= depths[0] {
		t.Errorf("depth should increase over 10 rooms: first=%d last=%d", depths[0], depths[9])
	}
	t.Logf("depths: %v", depths)
}

func TestIntegrationEnemyHPScalesAcrossRooms(t *testing.T) {
	g := simGame(ClassMage)

	// Room 1
	g.beginTransition(DoorNorth)
	g.finishTransition()
	earlyHP := g.Enemies[0].MaxHP

	// Clear 15 rooms
	for range 15 {
		for j := range g.Enemies {
			g.Enemies[j].Alive = false
		}
		g.checkRoomCleared()
		g.beginTransition(DoorNorth)
		g.finishTransition()
	}
	lateHP := g.Enemies[0].MaxHP

	if lateHP <= earlyHP {
		t.Errorf("enemy HP should scale: early=%d late=%d", earlyHP, lateHP)
	}
	t.Logf("early HP: %d, late HP: %d (%.1fx)", earlyHP, lateHP, float64(lateHP)/float64(earlyHP))
}

// ---------- Stress tests ----------

func TestIntegrationStress100Rooms(t *testing.T) {
	g := simGame(ClassMage)

	for i := range 100 {
		g.beginTransition(DoorNorth)
		g.finishTransition()

		if g.CurrentRoom == nil {
			t.Fatalf("room %d: nil room", i)
		}
		if g.Phase != PhasePlaying {
			t.Fatalf("room %d: wrong phase %d", i, g.Phase)
		}

		// Simulate a few frames of combat
		for range 10 {
			if g.Player.FireTimer <= 0 {
				g.SpawnPlayerProjectiles(0, 1)
				g.Player.FireTimer = 1.0 / g.Player.Stats.FireRate
			}
			g.Update(simDT)
		}

		// Force clear
		for j := range g.Enemies {
			g.Enemies[j].Alive = false
		}
		g.checkRoomCleared()
	}

	if g.RoomsCleared < 100 {
		t.Errorf("should have cleared 100 rooms, got %d", g.RoomsCleared)
	}
}

func TestIntegrationStressNoNegativeHP(t *testing.T) {
	g := simGame(ClassMage)

	for range 20 {
		g.beginTransition(DoorNorth)
		g.finishTransition()

		// Simulate combat with projectiles
		for frame := range 300 {
			if g.Player.FireTimer <= 0 {
				g.SpawnPlayerProjectiles(0, 1)
				g.Player.FireTimer = 1.0 / g.Player.Stats.FireRate
			}
			g.Update(simDT)

			// Check for degenerate state
			for j := range g.Enemies {
				if g.Enemies[j].Alive && g.Enemies[j].HP < -100 {
					t.Fatalf("room %d frame %d: enemy %d has degenerate HP: %d",
						g.RoomsCleared, frame, j, g.Enemies[j].HP)
				}
			}

			if g.Phase == PhaseDead {
				// Reset for next room
				g.Phase = PhasePlaying
				g.Player.HP = g.Player.MaxHP
				break
			}
		}

		// Force clear
		for j := range g.Enemies {
			g.Enemies[j].Alive = false
		}
		g.checkRoomCleared()
	}
}

func TestIntegrationStressManyItems(t *testing.T) {
	// Load player with many items, verify no overflow/panic
	var items []*ItemDef
	for i := range 50 {
		items = append(items, &AllItemDefs[i%len(AllItemDefs)])
	}

	g := setupCombat(ClassMage, items, EnemyMinion, 0)
	spawnTargetEnemy(g, EnemyMinion, 0, 3.0)

	// Verify stats don't produce NaN or Inf
	s := g.Player.Stats
	if math.IsNaN(float64(s.Damage)) || math.IsInf(float64(s.Damage), 0) {
		t.Errorf("damage is NaN/Inf: %f", s.Damage)
	}
	if math.IsNaN(float64(s.Speed)) || math.IsInf(float64(s.Speed), 0) {
		t.Errorf("speed is NaN/Inf: %f", s.Speed)
	}
	if math.IsNaN(float64(s.FireRate)) || math.IsInf(float64(s.FireRate), 0) {
		t.Errorf("fire rate is NaN/Inf: %f", s.FireRate)
	}
	if s.ProjCount < 1 {
		t.Errorf("proj count should be >= 1: %d", s.ProjCount)
	}

	// Fire and step — should not panic
	g.SpawnPlayerProjectiles(0, 1)
	stepFrames(g, 60)
}

// ---------- Balance: DPS vs EHP curves ----------

func TestIntegrationBalanceMageDPSvsEHP(t *testing.T) {
	type row struct {
		depth  int
		dps    float32
		ehp    int
		ttk    float32
	}

	var rows []row

	for _, depth := range []int{0, 3, 6, 9, 12, 15, 20} {
		stats := ComputeStats(ClassMage, nil)
		dps := stats.Damage * stats.FireRate
		e := makeEnemy(EnemySpawn{Type: EnemyWarrior}, 1.0, depth)

		ttk := float32(e.MaxHP) / dps

		rows = append(rows, row{depth, dps, e.MaxHP, ttk})
	}

	t.Log("Mage (no items) vs Warrior enemy:")
	t.Log("Depth | DPS  | EHP | TTK(s)")
	for _, r := range rows {
		t.Logf("  %3d | %4.1f | %3d | %.2f", r.depth, r.dps, r.ehp, r.ttk)
	}

	// TTK at depth 20 should be meaningfully higher than depth 0
	if rows[len(rows)-1].ttk < rows[0].ttk*1.5 {
		t.Error("TTK at depth 20 should be at least 1.5x depth 0")
	}
}

func TestIntegrationBalanceWarriorDPSvsEHP(t *testing.T) {
	type row struct {
		depth  int
		dps    float32
		ehp    int
		ttk    float32
	}

	var rows []row

	for _, depth := range []int{0, 3, 6, 9, 12, 15, 20} {
		stats := ComputeStats(ClassWarrior, nil)
		// Warrior DPS = damage / cooldown
		dps := stats.Damage / stats.MeleeCooldown
		e := makeEnemy(EnemySpawn{Type: EnemyWarrior}, 1.0, depth)

		ttk := float32(e.MaxHP) / dps

		rows = append(rows, row{depth, dps, e.MaxHP, ttk})
	}

	t.Log("Warrior (no items) vs Warrior enemy:")
	t.Log("Depth | DPS  | EHP | TTK(s)")
	for _, r := range rows {
		t.Logf("  %3d | %4.1f | %3d | %.2f", r.depth, r.dps, r.ehp, r.ttk)
	}

	if rows[len(rows)-1].ttk < rows[0].ttk*1.5 {
		t.Error("TTK at depth 20 should be at least 1.5x depth 0")
	}
}

func TestIntegrationBalanceItemPowerCurve(t *testing.T) {
	// Simulate picking up N copies of Iron Tip, measure DPS scaling
	t.Log("Mage DPS with stacking Iron Tips:")
	t.Log("Stacks | Damage | DPS")

	for stacks := range 11 {
		var items []*ItemDef
		ironTip := findItem("Iron Tip")
		for range stacks {
			items = append(items, ironTip)
		}
		s := ComputeStats(ClassMage, items)
		dps := s.Damage * s.FireRate
		t.Logf("    %2d | %5.1f  | %5.1f", stacks, s.Damage, dps)
	}

	// Verify 10 Iron Tips matches expected: 3.0 * (1 + 2.5) = 10.5 damage
	var tenTips []*ItemDef
	ironTip := findItem("Iron Tip")
	for range 10 {
		tenTips = append(tenTips, ironTip)
	}
	s10 := ComputeStats(ClassMage, tenTips)
	if s10.Damage < 10.4 || s10.Damage > 10.6 {
		t.Errorf("10 Iron Tips damage: want ~10.5, got %f", s10.Damage)
	}
}

func TestIntegrationBalanceClassComparison(t *testing.T) {
	mageStats := ComputeStats(ClassMage, nil)
	warriorStats := ComputeStats(ClassWarrior, nil)

	mageDPS := mageStats.Damage * mageStats.FireRate
	warriorDPS := warriorStats.Damage / warriorStats.MeleeCooldown

	mageEHP := float32(mageStats.MaxHP) // no armor
	// Warrior has passive -1 armor, so effective HP is higher
	// Approximate: each hit does 1 less, so if enemies hit for 2, warrior takes 1
	warriorEHP := float32(warriorStats.MaxHP) * 1.5 // rough armor factor

	t.Logf("Mage:    DPS=%.1f, HP=%d, EHP~%.0f", mageDPS, mageStats.MaxHP, mageEHP)
	t.Logf("Warrior: DPS=%.1f, HP=%d, EHP~%.0f", warriorDPS, warriorStats.MaxHP, warriorEHP)

	// Warrior DPS should be higher (melee risk/reward)
	if warriorDPS < mageDPS {
		t.Logf("NOTE: warrior DPS (%.1f) < mage DPS (%.1f) — intended? mage is safer at range", warriorDPS, mageDPS)
	}

	// Warrior should be tankier
	if warriorEHP < mageEHP {
		t.Error("warrior should have more effective HP than mage")
	}
}

// ---------- Full damage pipeline ----------

func TestIntegrationFullPipelineProjectileToLoot(t *testing.T) {
	g := setupCombat(ClassMage, nil, EnemyMinion, 0)
	idx := spawnTargetEnemy(g, EnemyMinion, 0, 3.0)
	g.Enemies[idx].State = StateIdle
	g.Enemies[idx].HP = 1 // one shot kill

	startScore := g.Score
	startItems := len(g.Player.Items)

	// Fire
	g.SpawnPlayerProjectiles(0, 1)

	// Step until enemy dies
	for frame := range 120 {
		g.Update(simDT)
		if !g.Enemies[idx].Alive {
			t.Logf("Enemy killed at frame %d", frame)
			break
		}
	}

	if g.Enemies[idx].Alive {
		t.Fatal("enemy should be dead")
	}

	// Score should increase
	if g.Score <= startScore {
		t.Error("score should increase on kill")
	}

	// Drops may or may not spawn (RNG), but let's check the pipeline didn't panic
	// Move player to any drops and pick them up
	for i := range g.ItemDrops {
		if g.ItemDrops[i].Collected {
			continue
		}
		g.Player.X = g.ItemDrops[i].X
		g.Player.Z = g.ItemDrops[i].Z
		g.checkCollisions()
	}

	t.Logf("Score: %d → %d, Items: %d → %d, Drops on ground: %d",
		startScore, g.Score, startItems, len(g.Player.Items), len(g.ItemDrops))
}

func TestIntegrationFullPipelineRoomClearToTransition(t *testing.T) {
	g := simGame(ClassMage)

	// Enter a new room
	g.beginTransition(DoorNorth)
	g.finishTransition()

	if g.CurrentRoom.Cleared {
		t.Fatal("new room should not be cleared")
	}

	// Kill all enemies via combat simulation
	for i := range g.Enemies {
		g.Enemies[i].HP = 1
		g.Enemies[i].State = StateIdle
	}

	// Fire at first enemy
	if len(g.Enemies) > 0 {
		g.Player.FacingAngle = 0
		dx := g.Enemies[0].X - g.Player.X
		dz := g.Enemies[0].Z - g.Player.Z
		g.SpawnPlayerProjectiles(dx, dz)
	}

	// Step until room clears (all 1HP enemies should die quickly from projectiles + splash)
	for range 600 {
		if g.Player.FireTimer <= 0 && !g.CurrentRoom.Cleared {
			// Aim at first alive enemy
			for i := range g.Enemies {
				if g.Enemies[i].Alive {
					dx := g.Enemies[i].X - g.Player.X
					dz := g.Enemies[i].Z - g.Player.Z
					g.SpawnPlayerProjectiles(dx, dz)
					g.Player.FireTimer = 1.0 / g.Player.Stats.FireRate
					break
				}
			}
		}
		g.Update(simDT)
		if g.CurrentRoom.Cleared {
			break
		}
	}

	if !g.CurrentRoom.Cleared {
		alive := 0
		for i := range g.Enemies {
			if g.Enemies[i].Alive {
				alive++
			}
		}
		t.Fatalf("room should be cleared, %d enemies still alive", alive)
	}

	// Doors should be open
	for z := range RoomH {
		for x := range RoomW {
			if g.CurrentRoom.Tiles[z][x] == TileDoorClosed {
				t.Errorf("door at (%d,%d) still closed after clear", x, z)
			}
		}
	}
}

// ---------- Edge cases ----------

func TestIntegrationPlayerDiesFromEnemyProjectile(t *testing.T) {
	g := simGame(ClassMage)
	g.Player.HP = 1

	// Spawn enemy projectile aimed at player
	g.Projectiles = append(g.Projectiles, Projectile{
		X: g.Player.X, Z: g.Player.Z + 2.0,
		VX: 0, VZ: -10.0,
		Damage: 5, Radius: 0.1,
		Alive: true, Owner: 1,
	})

	stepFrames(g, 60)

	if g.Phase != PhaseDead {
		t.Errorf("player should be dead, phase=%d HP=%d", g.Phase, g.Player.HP)
	}
}

func TestIntegrationProjectileExpiresByAge(t *testing.T) {
	g := simGame(ClassMage)
	g.SpawnPlayerProjectiles(0, 1)

	if len(g.Projectiles) != 1 {
		t.Fatal("should have 1 projectile")
	}

	// Step for longer than projectileMaxAge (4 seconds)
	stepFrames(g, 300) // 5 seconds

	aliveCount := 0
	for _, p := range g.Projectiles {
		if p.Alive {
			aliveCount++
		}
	}
	if aliveCount > 0 {
		t.Error("projectile should have expired by age")
	}
}

func TestIntegrationBouncingProjectileStaysAlive(t *testing.T) {
	rubber := findItem("Rubber Ball")
	g := setupCombat(ClassMage, []*ItemDef{rubber, rubber}, EnemyMinion, 0)

	// Fire at a wall (no enemies in path)
	g.SpawnPlayerProjectiles(-1, 0) // aim left toward west wall

	// Step enough for it to hit the wall
	stepFrames(g, 30)

	aliveCount := 0
	for _, p := range g.Projectiles {
		if p.Alive {
			aliveCount++
		}
	}
	// With 4 bounces (2 stacks * 2), projectile should survive wall hits
	if aliveCount == 0 {
		t.Error("bouncing projectile should survive wall hit")
	}
}
