package main

import (
	"math"
	"testing"
)

// helper: create a minimal game state for testing
func testGame(class PlayerClass) *GameState {
	g := NewGame(42, 1.0, 0.0, class)
	return g
}

// helper: place an enemy directly into the game
func addEnemy(g *GameState, etype EnemyType, x, z float32, depth int) int {
	e := makeEnemy(EnemySpawn{X: x, Z: z, Type: etype}, g.TileUnit, depth)
	g.Enemies = append(g.Enemies, e)
	return len(g.Enemies) - 1
}

// ---------- NewGame ----------

func TestNewGameMageInit(t *testing.T) {
	g := testGame(ClassMage)
	if g.Phase != PhasePlaying {
		t.Errorf("phase: want PhasePlaying, got %d", g.Phase)
	}
	if g.Player.HP != 6 {
		t.Errorf("mage HP: want 6, got %d", g.Player.HP)
	}
	if g.Player.MaxHP != 6 {
		t.Errorf("mage MaxHP: want 6, got %d", g.Player.MaxHP)
	}
	if g.Player.Class != ClassMage {
		t.Errorf("class: want ClassMage, got %d", g.Player.Class)
	}
	if g.CurrentRoom == nil {
		t.Fatal("current room is nil")
	}
	if !g.CurrentRoom.Cleared {
		t.Error("start room should be cleared")
	}
}

func TestNewGameWarriorInit(t *testing.T) {
	g := testGame(ClassWarrior)
	if g.Player.HP != 10 {
		t.Errorf("warrior HP: want 10, got %d", g.Player.HP)
	}
	if g.Player.Stats.MeleeArc != 120 {
		t.Errorf("warrior melee arc: want 120, got %f", g.Player.Stats.MeleeArc)
	}
}

func TestNewGamePlayerAtCenter(t *testing.T) {
	g := NewGame(42, 2.0, 0.0, ClassMage)
	// center = RoomW/2 * tileUnit, RoomH/2 * tileUnit
	wantX := float32(RoomW) / 2.0 * 2.0
	wantZ := float32(RoomH) / 2.0 * 2.0
	if g.Player.X != wantX || g.Player.Z != wantZ {
		t.Errorf("player pos: want (%f,%f), got (%f,%f)", wantX, wantZ, g.Player.X, g.Player.Z)
	}
}

func TestNewGameStartRoomInMap(t *testing.T) {
	g := testGame(ClassMage)
	room, ok := g.RoomMap[RoomCoord{0, 0}]
	if !ok {
		t.Fatal("start room not in map")
	}
	if room != g.CurrentRoom {
		t.Error("CurrentRoom doesn't match map entry")
	}
}

func TestNewGameDeterministic(t *testing.T) {
	g1 := NewGame(123, 1.0, 0.0, ClassMage)
	g2 := NewGame(123, 1.0, 0.0, ClassMage)
	if g1.Player.X != g2.Player.X || g1.Player.Z != g2.Player.Z {
		t.Error("same seed should produce same player position")
	}
	if g1.CurrentRoom.Tiles != g2.CurrentRoom.Tiles {
		t.Error("same seed should produce same room")
	}
}

// ---------- makeEnemy ----------

func TestMakeEnemyMinion(t *testing.T) {
	e := makeEnemy(EnemySpawn{X: 5.5, Z: 3.5, Type: EnemyMinion}, 2.0, 0)
	if e.X != 11.0 || e.Z != 7.0 {
		t.Errorf("position: want (11,7), got (%f,%f)", e.X, e.Z)
	}
	if e.HP != 6 || e.MaxHP != 6 {
		t.Errorf("minion HP at depth 0: want 6, got %d", e.HP)
	}
	if !e.Alive {
		t.Error("enemy should be alive")
	}
	if e.Speed != 3.5 {
		t.Errorf("minion speed: want 3.5, got %f", e.Speed)
	}
}

func TestMakeEnemyWarriorStats(t *testing.T) {
	e := makeEnemy(EnemySpawn{X: 5, Z: 5, Type: EnemyWarrior}, 1.0, 0)
	if e.HP != 12 {
		t.Errorf("warrior HP: want 12, got %d", e.HP)
	}
	if e.Damage != 2 {
		t.Errorf("warrior damage: want 2, got %d", e.Damage)
	}
}

func TestMakeEnemyMageStats(t *testing.T) {
	e := makeEnemy(EnemySpawn{X: 5, Z: 5, Type: EnemyMage}, 1.0, 0)
	if e.HP != 5 {
		t.Errorf("mage HP: want 5, got %d", e.HP)
	}
	if e.AttackRange != 6.0 {
		t.Errorf("mage attack range: want 6.0, got %f", e.AttackRange)
	}
}

func TestMakeEnemyHPScaling(t *testing.T) {
	// depth 10: scale = 1.0 + 10*0.12 = 2.2
	e0 := makeEnemy(EnemySpawn{Type: EnemyMinion}, 1.0, 0)
	e10 := makeEnemy(EnemySpawn{Type: EnemyMinion}, 1.0, 10)
	// 6 * 2.2 = 13.2 → 13
	if e10.HP <= e0.HP {
		t.Errorf("HP should scale with depth: depth0=%d depth10=%d", e0.HP, e10.HP)
	}
	scale := 1.0 + float64(10)*0.12
	wantHP := int(float64(6) * scale)
	if e10.HP != wantHP {
		t.Errorf("depth 10 minion HP: want %d, got %d", wantHP, e10.HP)
	}
}

func TestMakeEnemyHPScalingAllTypes(t *testing.T) {
	for _, etype := range []EnemyType{EnemyMinion, EnemyWarrior, EnemyMage} {
		e0 := makeEnemy(EnemySpawn{Type: etype}, 1.0, 0)
		e20 := makeEnemy(EnemySpawn{Type: etype}, 1.0, 20)
		if e20.HP <= e0.HP {
			t.Errorf("type %d: HP should increase with depth (%d vs %d)", etype, e0.HP, e20.HP)
		}
		if e20.HP != e20.MaxHP {
			t.Errorf("type %d: HP should equal MaxHP on creation", etype)
		}
	}
}

// ---------- damagePlayer ----------

func TestDamagePlayerBasic(t *testing.T) {
	g := testGame(ClassMage)
	startHP := g.Player.HP
	g.damagePlayer(2)
	if g.Player.HP != startHP-2 {
		t.Errorf("HP after 2 damage: want %d, got %d", startHP-2, g.Player.HP)
	}
}

func TestDamagePlayerInvulnerable(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.InvulnTimer = 1.0
	startHP := g.Player.HP
	g.damagePlayer(5)
	if g.Player.HP != startHP {
		t.Errorf("should not take damage while invulnerable: want %d, got %d", startHP, g.Player.HP)
	}
}

func TestDamagePlayerDodging(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.DodgeTimer = 0.5
	startHP := g.Player.HP
	g.damagePlayer(5)
	if g.Player.HP != startHP {
		t.Errorf("should not take damage while dodging: want %d, got %d", startHP, g.Player.HP)
	}
}

func TestDamagePlayerShieldAbsorb(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.Stats.ShieldStacks = 2
	startHP := g.Player.HP
	g.damagePlayer(5)
	if g.Player.HP != startHP {
		t.Error("shield should absorb damage completely")
	}
	if g.Player.Stats.ShieldStacks != 1 {
		t.Errorf("shield stacks: want 1, got %d", g.Player.Stats.ShieldStacks)
	}
	// Second hit
	g.Player.InvulnTimer = 0 // reset invuln from shield
	g.damagePlayer(5)
	if g.Player.Stats.ShieldStacks != 0 {
		t.Errorf("shield stacks: want 0, got %d", g.Player.Stats.ShieldStacks)
	}
	if g.Player.HP != startHP {
		t.Error("second shield should also absorb")
	}
}

func TestDamagePlayerWarriorParry(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.ParryWindow = 0.1
	g.Player.BlockTimer = 0.5
	startHP := g.Player.HP
	g.damagePlayer(5)
	if g.Player.HP != startHP {
		t.Error("parry should negate all damage")
	}
	if g.Player.ParryWindow != 0 {
		t.Error("parry window should be consumed")
	}
}

func TestDamagePlayerWarriorBlock(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.BlockTimer = 0.5
	g.Player.ParryWindow = 0 // no parry
	startHP := g.Player.HP
	g.damagePlayer(4)
	// Block halves: (4+1)/2 = 2
	expected := startHP - 2
	if g.Player.HP != expected {
		t.Errorf("block should halve damage: want HP=%d, got %d", expected, g.Player.HP)
	}
}

func TestDamagePlayerWarriorBlockOddDamage(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.BlockTimer = 0.5
	g.Player.ParryWindow = 0
	startHP := g.Player.HP
	g.damagePlayer(3)
	// (3+1)/2 = 2
	expected := startHP - 2
	if g.Player.HP != expected {
		t.Errorf("block odd damage: want HP=%d, got %d", expected, g.Player.HP)
	}
}

func TestDamagePlayerWarriorBlockMinOneDamage(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.BlockTimer = 0.5
	g.Player.ParryWindow = 0
	startHP := g.Player.HP
	g.damagePlayer(1)
	// (1+1)/2 = 1
	expected := startHP - 1
	if g.Player.HP != expected {
		t.Errorf("block min damage: want HP=%d, got %d", expected, g.Player.HP)
	}
}

func TestDamagePlayerWarriorPassiveArmor(t *testing.T) {
	g := testGame(ClassWarrior)
	startHP := g.Player.HP
	g.damagePlayer(3)
	// Passive armor: dmg > 1 → dmg-1 = 2
	expected := startHP - 2
	if g.Player.HP != expected {
		t.Errorf("passive armor: want HP=%d, got %d", expected, g.Player.HP)
	}
}

func TestDamagePlayerWarriorPassiveArmorMinOne(t *testing.T) {
	g := testGame(ClassWarrior)
	startHP := g.Player.HP
	g.damagePlayer(1)
	// dmg == 1: no reduction
	expected := startHP - 1
	if g.Player.HP != expected {
		t.Errorf("passive armor should not reduce 1 damage: want HP=%d, got %d", expected, g.Player.HP)
	}
}

func TestDamagePlayerThorns(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.Stats.ThornsStacks = 2
	idx := addEnemy(g, EnemyMinion, g.Player.X+0.5, g.Player.Z, 0)
	startEHP := g.Enemies[idx].HP
	g.damagePlayer(2)
	// thorns = 2 * 2 = 4 damage to nearby enemies
	if g.Enemies[idx].HP != startEHP-4 {
		t.Errorf("thorns damage: want %d, got %d", startEHP-4, g.Enemies[idx].HP)
	}
}

func TestDamagePlayerThornsOutOfRange(t *testing.T) {
	g := testGame(ClassWarrior)
	g.Player.Stats.ThornsStacks = 2
	// Place enemy far away (> 2.5 * tileUnit)
	idx := addEnemy(g, EnemyMinion, g.Player.X+5.0, g.Player.Z, 0)
	startEHP := g.Enemies[idx].HP
	g.damagePlayer(2)
	if g.Enemies[idx].HP != startEHP {
		t.Error("thorns should not hit enemies out of range")
	}
}

func TestDamagePlayerSetsInvuln(t *testing.T) {
	g := testGame(ClassMage)
	g.damagePlayer(1)
	if g.Player.InvulnTimer <= 0 {
		t.Error("taking damage should grant invulnerability frames")
	}
}

func TestDamagePlayerSetsTimeSinceHit(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.TimeSinceHit = 10.0
	g.damagePlayer(1)
	if g.Player.TimeSinceHit != 0 {
		t.Errorf("TimeSinceHit should reset to 0, got %f", g.Player.TimeSinceHit)
	}
}

// ---------- killEnemy ----------

func TestKillEnemyMarksNotAlive(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.killEnemy(0)
	if g.Enemies[0].Alive {
		t.Error("enemy should be dead")
	}
}

func TestKillEnemyDeathTimer(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.killEnemy(0)
	if g.Enemies[0].DeathTimer != 1.0 {
		t.Errorf("death timer: want 1.0, got %f", g.Enemies[0].DeathTimer)
	}
}

func TestKillEnemyScoring(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	addEnemy(g, EnemyWarrior, 6, 5, 0)
	addEnemy(g, EnemyMage, 7, 5, 0)

	g.killEnemy(0)
	// Minion: 10 * (0 + 1) = 10
	if g.Score != 10 {
		t.Errorf("minion score: want 10, got %d", g.Score)
	}
	g.killEnemy(1)
	// Warrior: 10 * (1 + 1) = 20, total 30
	if g.Score != 30 {
		t.Errorf("warrior score: want 30, got %d", g.Score)
	}
	g.killEnemy(2)
	// Mage: 10 * (2 + 1) = 30, total 60
	if g.Score != 60 {
		t.Errorf("mage score: want 60, got %d", g.Score)
	}
}

func TestKillEnemyEmitsSFX(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.SFXQueue = nil
	g.killEnemy(0)
	found := false
	for _, sfx := range g.SFXQueue {
		if sfx == SFXEnemyDeath {
			found = true
		}
	}
	if !found {
		t.Error("killEnemy should emit SFXEnemyDeath")
	}
}

// ---------- checkRoomCleared ----------

func TestCheckRoomClearedSkipsAlreadyCleared(t *testing.T) {
	g := testGame(ClassMage)
	// Start room is already cleared
	cleared := g.RoomsCleared
	g.checkRoomCleared()
	if g.RoomsCleared != cleared {
		t.Error("should not re-count already cleared room")
	}
}

func TestCheckRoomClearedNotClearedWithAlive(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Cleared = false
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.checkRoomCleared()
	if g.CurrentRoom.Cleared {
		t.Error("room should not be cleared with alive enemy")
	}
}

func TestCheckRoomClearedAllDead(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Cleared = false
	// Place closed doors
	for d := DoorDir(0); d < DoorCount; d++ {
		if g.CurrentRoom.Doors[d] {
			for _, tile := range doorTiles(d) {
				g.CurrentRoom.Tiles[tile[1]][tile[0]] = TileDoorClosed
			}
		}
	}
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].Alive = false
	startHP := g.Player.HP
	g.Player.HP = g.Player.MaxHP - 1 // ensure heal is testable

	g.checkRoomCleared()

	if !g.CurrentRoom.Cleared {
		t.Error("room should be cleared when all enemies dead")
	}
	if g.Player.HP != g.Player.MaxHP-1+1 {
		t.Errorf("should heal 1 HP on clear: want %d, got %d", startHP, g.Player.HP)
	}
}

func TestCheckRoomClearedOpensDoors(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Cleared = false
	// Set all doors to closed
	for d := DoorDir(0); d < DoorCount; d++ {
		if g.CurrentRoom.Doors[d] {
			for _, tile := range doorTiles(d) {
				g.CurrentRoom.Tiles[tile[1]][tile[0]] = TileDoorClosed
			}
		}
	}
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].Alive = false

	g.checkRoomCleared()

	for z := 0; z < RoomH; z++ {
		for x := 0; x < RoomW; x++ {
			if g.CurrentRoom.Tiles[z][x] == TileDoorClosed {
				t.Errorf("door at (%d,%d) still closed after clear", x, z)
			}
		}
	}
}

func TestCheckRoomClearedHealCappedAtMax(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Cleared = false
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].Alive = false
	g.Player.HP = g.Player.MaxHP // already full

	g.checkRoomCleared()

	if g.Player.HP != g.Player.MaxHP {
		t.Errorf("HP should not exceed max: want %d, got %d", g.Player.MaxHP, g.Player.HP)
	}
}

func TestCheckRoomClearedShieldReset(t *testing.T) {
	g := testGame(ClassMage)
	// Give player a shield item
	shield := findItem("Barrier Gem")
	g.Player.Items = []*ItemDef{shield}
	g.Player.Stats = ComputeStats(ClassMage, g.Player.Items)
	g.Player.Stats.ShieldStacks = 0 // used up

	g.CurrentRoom.Cleared = false
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].Alive = false

	g.checkRoomCleared()

	if g.Player.Stats.ShieldStacks != 1 {
		t.Errorf("shields should reset on clear: want 1, got %d", g.Player.Stats.ShieldStacks)
	}
}

// ---------- isTileWalkable ----------

func TestIsTileWalkableFloor(t *testing.T) {
	g := testGame(ClassMage)
	// Interior tile should be floor
	if !g.isTileWalkable(5, 5) {
		t.Error("interior floor tile should be walkable")
	}
}

func TestIsTileWalkableWall(t *testing.T) {
	g := testGame(ClassMage)
	if g.isTileWalkable(0, 0) {
		t.Error("corner wall should not be walkable")
	}
}

func TestIsTileWalkableOutOfBounds(t *testing.T) {
	g := testGame(ClassMage)
	if g.isTileWalkable(-1, 5) {
		t.Error("negative x should not be walkable")
	}
	if g.isTileWalkable(5, -1) {
		t.Error("negative z should not be walkable")
	}
	if g.isTileWalkable(RoomW, 5) {
		t.Error("x >= RoomW should not be walkable")
	}
	if g.isTileWalkable(5, RoomH) {
		t.Error("z >= RoomH should not be walkable")
	}
}

func TestIsTileWalkableClosedDoor(t *testing.T) {
	g := testGame(ClassMage)
	// Set a door tile to closed
	tiles := doorTiles(DoorNorth)
	g.CurrentRoom.Tiles[tiles[0][1]][tiles[0][0]] = TileDoorClosed
	if g.isTileWalkable(tiles[0][0], tiles[0][1]) {
		t.Error("closed door should not be walkable")
	}
}

func TestIsTileWalkableOpenDoor(t *testing.T) {
	g := testGame(ClassMage)
	tiles := doorTiles(DoorNorth)
	g.CurrentRoom.Tiles[tiles[0][1]][tiles[0][0]] = TileDoorOpen
	if !g.isTileWalkable(tiles[0][0], tiles[0][1]) {
		t.Error("open door should be walkable")
	}
}

func TestIsTileWalkableBlockingProp(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Props[[2]int{5, 5}] = Prop{Blocking: true}
	if g.isTileWalkable(5, 5) {
		t.Error("blocking prop should not be walkable")
	}
}

func TestIsTileWalkableNonBlockingProp(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom.Props[[2]int{5, 5}] = Prop{Blocking: false}
	if !g.isTileWalkable(5, 5) {
		t.Error("non-blocking prop on floor should be walkable")
	}
}

func TestIsTileWalkableNilRoom(t *testing.T) {
	g := testGame(ClassMage)
	g.CurrentRoom = nil
	if g.isTileWalkable(5, 5) {
		t.Error("nil room should not be walkable")
	}
}

// ---------- resolveWallCollision ----------

func TestResolveWallCollisionNoCollision(t *testing.T) {
	g := testGame(ClassMage)
	// Center of room, should not be pushed
	x, z := float32(7.5), float32(5.5)
	rx, rz := g.resolveWallCollision(x, z, 0.35)
	if rx != x || rz != z {
		t.Errorf("center should not be pushed: (%f,%f) → (%f,%f)", x, z, rx, rz)
	}
}

func TestResolveWallCollisionPushesFromWall(t *testing.T) {
	g := testGame(ClassMage)
	// Use z=2.5 to avoid the west door (centered at z=RoomH/2=5)
	// tile (0, 2) is a wall since it's not a door tile
	x, z := float32(0.8), float32(2.5)
	rx, _ := g.resolveWallCollision(x, z, 0.35)
	if rx <= x {
		t.Errorf("should push away from left wall: %f → %f", x, rx)
	}
}

// ---------- alertNearby / alertAllInRadius ----------

func TestAlertNearby(t *testing.T) {
	g := testGame(ClassMage)
	idx0 := addEnemy(g, EnemyMinion, 5, 5, 0)
	idx1 := addEnemy(g, EnemyMinion, 5.5, 5, 0)
	idx2 := addEnemy(g, EnemyMinion, 20, 20, 0) // far away

	g.Enemies[idx0].State = StateChasing
	g.Enemies[idx1].State = StateIdle
	g.Enemies[idx2].State = StateIdle

	g.alertNearby(idx0, 3.0)

	if g.Enemies[idx1].State != StateChasing {
		t.Error("nearby idle enemy should be alerted")
	}
	if g.Enemies[idx2].State != StateIdle {
		t.Error("far away enemy should remain idle")
	}
}

func TestAlertAllInRadius(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	addEnemy(g, EnemyMinion, 5.5, 5, 0)
	addEnemy(g, EnemyMinion, 50, 50, 0)

	g.Enemies[0].State = StateIdle
	g.Enemies[1].State = StateIdle
	g.Enemies[2].State = StateIdle

	g.alertAllInRadius(5.0, 5.0, 2.0)

	if g.Enemies[0].State != StateChasing {
		t.Error("enemy 0 should be alerted")
	}
	if g.Enemies[1].State != StateChasing {
		t.Error("enemy 1 should be alerted")
	}
	if g.Enemies[2].State != StateIdle {
		t.Error("enemy 2 should remain idle")
	}
}

func TestAlertNearbySkipsDeadEnemies(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	addEnemy(g, EnemyMinion, 5.5, 5, 0)
	g.Enemies[0].State = StateChasing
	g.Enemies[1].State = StateIdle
	g.Enemies[1].Alive = false

	g.alertNearby(0, 3.0)

	if g.Enemies[1].State != StateIdle {
		t.Error("dead enemy should not be alerted")
	}
}

// ---------- staggerNearby ----------

func TestStaggerNearby(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, g.Player.X+0.5, g.Player.Z, 0)
	addEnemy(g, EnemyMinion, g.Player.X+10, g.Player.Z, 0) // far

	origSpeed := g.Enemies[0].Speed

	g.staggerNearby(g.Player.X, g.Player.Z, 2.0)

	if g.Enemies[0].AttackTimer != 1.5 {
		t.Errorf("staggered attack timer: want 1.5, got %f", g.Enemies[0].AttackTimer)
	}
	if g.Enemies[0].Speed >= origSpeed {
		t.Error("staggered enemy should be slowed")
	}
	if g.Enemies[1].AttackTimer == 1.5 {
		t.Error("far enemy should not be staggered")
	}
}

// ---------- SpawnPlayerProjectiles ----------

func TestSpawnPlayerProjectilesMageBasic(t *testing.T) {
	g := testGame(ClassMage)
	g.SpawnPlayerProjectiles(0, 1) // aim forward
	if len(g.Projectiles) != 1 {
		t.Fatalf("want 1 projectile, got %d", len(g.Projectiles))
	}
	p := g.Projectiles[0]
	if p.Owner != 0 {
		t.Error("projectile owner should be player (0)")
	}
	if !p.Alive {
		t.Error("projectile should be alive")
	}
	if p.Damage != 3.0 {
		t.Errorf("base mage projectile damage: want 3.0, got %f", p.Damage)
	}
}

func TestSpawnPlayerProjectilesWithSplit(t *testing.T) {
	g := testGame(ClassMage)
	prism := findItem("Prism Lens")
	g.Player.Items = []*ItemDef{prism}
	g.Player.Stats = ComputeStats(ClassMage, g.Player.Items)

	g.SpawnPlayerProjectiles(0, 1)
	if len(g.Projectiles) != 3 {
		t.Errorf("with 1 split, want 3 projectiles, got %d", len(g.Projectiles))
	}
}

func TestSpawnPlayerProjectilesWithTags(t *testing.T) {
	g := testGame(ClassMage)
	flame := findItem("Flame Shard")
	ghost := findItem("Ghost Arrow")
	rubber := findItem("Rubber Ball")
	g.Player.Items = []*ItemDef{flame, ghost, rubber}
	g.Player.Stats = ComputeStats(ClassMage, g.Player.Items)

	g.SpawnPlayerProjectiles(1, 0)
	p := g.Projectiles[0]

	if !p.Fire {
		t.Error("should have fire tag")
	}
	if !p.Pierce {
		t.Error("should have pierce tag")
	}
	if p.PierceLeft != 1 {
		t.Errorf("pierce left: want 1, got %d", p.PierceLeft)
	}
	if !p.Bounce {
		t.Error("should have bounce tag")
	}
	if p.BounceLeft != 2 {
		t.Errorf("bounce left: want 2, got %d", p.BounceLeft)
	}
}

func TestSpawnPlayerProjectilesSpread(t *testing.T) {
	g := testGame(ClassMage)
	prism := findItem("Prism Lens")
	g.Player.Items = []*ItemDef{prism, prism}
	g.Player.Stats = ComputeStats(ClassMage, g.Player.Items)

	g.SpawnPlayerProjectiles(0, 1) // aim Z+
	// 5 projectiles, each with slightly different angle
	if len(g.Projectiles) != 5 {
		t.Fatalf("want 5 projectiles, got %d", len(g.Projectiles))
	}
	// Middle projectile should aim most forward (highest VZ)
	midVZ := g.Projectiles[2].VZ
	for i, p := range g.Projectiles {
		if i == 2 {
			continue
		}
		if p.VZ > midVZ+0.001 {
			t.Errorf("projectile %d VZ=%f exceeds center VZ=%f", i, p.VZ, midVZ)
		}
	}
}

// ---------- beginTransition / finishTransition ----------

func TestBeginTransition(t *testing.T) {
	g := testGame(ClassMage)
	g.beginTransition(DoorNorth)
	if g.Phase != PhaseTransition {
		t.Error("phase should be PhaseTransition")
	}
	if g.TransitionTimer != 0.5 {
		t.Errorf("transition timer: want 0.5, got %f", g.TransitionTimer)
	}
	if g.TransitionDir != DoorNorth {
		t.Errorf("transition dir: want DoorNorth, got %d", g.TransitionDir)
	}
}

func TestFinishTransitionCreatesNewRoom(t *testing.T) {
	g := testGame(ClassMage)
	g.beginTransition(DoorNorth)
	g.finishTransition()

	expected := RoomCoord{0, -1}
	if g.RoomCoord != expected {
		t.Errorf("room coord: want %v, got %v", expected, g.RoomCoord)
	}
	if _, ok := g.RoomMap[expected]; !ok {
		t.Error("new room should be in map")
	}
	if g.Phase != PhasePlaying {
		t.Error("phase should return to PhasePlaying")
	}
	if len(g.Enemies) == 0 {
		t.Error("new room should have enemies")
	}
}

func TestFinishTransitionReusesExistingRoom(t *testing.T) {
	g := testGame(ClassMage)
	// Go north
	g.beginTransition(DoorNorth)
	g.finishTransition()
	northRoom := g.CurrentRoom

	// Go back south
	g.CurrentRoom.Cleared = true // force clear to allow transition
	g.beginTransition(DoorSouth)
	g.finishTransition()

	// Go north again
	g.beginTransition(DoorNorth)
	g.finishTransition()

	if g.CurrentRoom != northRoom {
		t.Error("should reuse existing room, not create new one")
	}
}

func TestFinishTransitionClearsProjectiles(t *testing.T) {
	g := testGame(ClassMage)
	g.Projectiles = append(g.Projectiles, Projectile{Alive: true})
	g.ItemDrops = append(g.ItemDrops, ItemDrop{})
	g.Explosions = append(g.Explosions, Explosion{})

	g.beginTransition(DoorNorth)
	g.finishTransition()

	if len(g.Projectiles) != 0 {
		t.Error("projectiles should be cleared on transition")
	}
	if len(g.ItemDrops) != 0 {
		t.Error("item drops should be cleared on transition")
	}
	if len(g.Explosions) != 0 {
		t.Error("explosions should be cleared on transition")
	}
}

// ---------- Update phase transitions ----------

func TestUpdateTransitionCountdown(t *testing.T) {
	g := testGame(ClassMage)
	g.Phase = PhaseTransition
	g.TransitionTimer = 0.5
	g.TransitionDir = DoorNorth
	g.Update(0.3)
	if g.TransitionTimer > 0.21 || g.TransitionTimer < 0.19 {
		t.Errorf("timer should be ~0.2, got %f", g.TransitionTimer)
	}
}

func TestUpdateDeathTimer(t *testing.T) {
	g := testGame(ClassMage)
	g.Phase = PhaseDead
	g.DeathTimer = 1.5
	g.Update(0.5)
	if g.DeathTimer > 1.01 || g.DeathTimer < 0.99 {
		t.Errorf("death timer should be ~1.0, got %f", g.DeathTimer)
	}
}

func TestUpdatePlayerDeath(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.HP = 0
	g.Update(1.0 / 60.0)
	if g.Phase != PhaseDead {
		t.Error("should transition to PhaseDead when HP <= 0")
	}
}

// ---------- spawnExplosion ----------

func TestSpawnExplosion(t *testing.T) {
	g := testGame(ClassMage)
	g.spawnExplosion(5.0, 5.0, 10.0, 0)
	if len(g.Explosions) != 1 {
		t.Fatalf("want 1 explosion, got %d", len(g.Explosions))
	}
	exp := g.Explosions[0]
	if exp.X != 5.0 || exp.Z != 5.0 {
		t.Error("explosion position wrong")
	}
	if exp.Damage != 10.0 {
		t.Errorf("explosion damage: want 10.0, got %f", exp.Damage)
	}
	if exp.Timer != 0.3 {
		t.Errorf("explosion timer: want 0.3, got %f", exp.Timer)
	}
}

func TestSpawnExplosionRadiusScalesWithStacks(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.Stats.ExplosiveStacks = 0
	g.spawnExplosion(5, 5, 10, 0)
	r0 := g.Explosions[0].Radius

	g.Player.Stats.ExplosiveStacks = 3
	g.spawnExplosion(5, 5, 10, 0)
	r3 := g.Explosions[1].Radius

	if r3 <= r0 {
		t.Errorf("higher explosive stacks should increase radius: %f vs %f", r0, r3)
	}
}

// ---------- Status effects on enemies ----------

func TestEnemyFireDamageOverTime(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyWarrior, 5, 5, 0)
	g.Enemies[0].FireTimer = 1.0
	g.Enemies[0].FireDamage = 3.0
	startHP := g.Enemies[0].HP

	g.updateEnemies(0.5)

	// ceil(3.0 * 0.5) = 2 damage
	expectedDmg := int(math.Ceil(3.0 * 0.5))
	if g.Enemies[0].HP != startHP-expectedDmg {
		t.Errorf("fire DoT: want HP=%d, got %d", startHP-expectedDmg, g.Enemies[0].HP)
	}
}

func TestEnemyIceSlow(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].State = StateChasing
	g.Enemies[0].IceTimer = 2.0
	startX := g.Enemies[0].X

	// Ensure player is far enough to trigger chasing movement
	g.Player.X = 10.0
	g.Player.Z = 5.0

	g.updateEnemies(0.1)
	icedDist := g.Enemies[0].X - startX

	// Reset and try without ice
	g.Enemies[0].X = startX
	g.Enemies[0].IceTimer = 0
	g.updateEnemies(0.1)
	normalDist := g.Enemies[0].X - startX

	if icedDist >= normalDist {
		t.Errorf("iced enemy should move slower: iced=%f normal=%f", icedDist, normalDist)
	}
}

func TestEnemyPoisonDamage(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyWarrior, 5, 5, 0)
	g.Enemies[0].PoisonTimer = 2.0
	g.Enemies[0].PoisonDmg = 2.0
	startHP := g.Enemies[0].HP

	g.updateEnemies(0.5)

	expectedDmg := int(math.Ceil(2.0 * 0.5))
	if g.Enemies[0].HP != startHP-expectedDmg {
		t.Errorf("poison DoT: want HP=%d, got %d", startHP-expectedDmg, g.Enemies[0].HP)
	}
}

func TestEnemyCorneredSpeedBoost(t *testing.T) {
	g := testGame(ClassMage)
	addEnemy(g, EnemyMinion, 5, 5, 0)
	g.Enemies[0].State = StateChasing
	g.Player.X = 10.0
	g.Player.Z = 5.0

	// Normal: full HP
	startX := g.Enemies[0].X
	g.updateEnemies(0.1)
	normalDist := g.Enemies[0].X - startX

	// Cornered: <= 25% HP
	g.Enemies[0].X = startX
	g.Enemies[0].HP = 1
	g.Enemies[0].MaxHP = 6
	g.updateEnemies(0.1)
	corneredDist := g.Enemies[0].X - startX

	if corneredDist <= normalDist {
		t.Errorf("cornered enemy should move faster: normal=%f cornered=%f", normalDist, corneredDist)
	}
}

// ---------- healthPickup ----------

func TestHealthPickupHeals(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.HP = 3
	g.Player.MaxHP = 6
	g.ItemDrops = append(g.ItemDrops, ItemDrop{
		X: g.Player.X, Z: g.Player.Z, Item: &healthPickup,
	})
	g.checkCollisions()
	if g.Player.HP != 5 {
		t.Errorf("health pickup: want HP=5, got %d", g.Player.HP)
	}
}

func TestHealthPickupCapsAtMax(t *testing.T) {
	g := testGame(ClassMage)
	g.Player.HP = 5
	g.Player.MaxHP = 6
	g.ItemDrops = append(g.ItemDrops, ItemDrop{
		X: g.Player.X, Z: g.Player.Z, Item: &healthPickup,
	})
	g.checkCollisions()
	if g.Player.HP != 6 {
		t.Errorf("health pickup cap: want HP=6, got %d", g.Player.HP)
	}
}

func TestItemPickupAddsToInventory(t *testing.T) {
	g := testGame(ClassMage)
	item := findItem("Iron Tip")
	g.ItemDrops = append(g.ItemDrops, ItemDrop{
		X: g.Player.X, Z: g.Player.Z, Item: item,
	})
	g.checkCollisions()
	if len(g.Player.Items) != 1 {
		t.Fatalf("should have 1 item, got %d", len(g.Player.Items))
	}
	if g.Player.Items[0].Name != "Iron Tip" {
		t.Errorf("picked up wrong item: %s", g.Player.Items[0].Name)
	}
}

func TestItemPickupUpdatesStats(t *testing.T) {
	g := testGame(ClassMage)
	item := findItem("Iron Tip")
	oldDmg := g.Player.Stats.Damage
	g.ItemDrops = append(g.ItemDrops, ItemDrop{
		X: g.Player.X, Z: g.Player.Z, Item: item,
	})
	g.checkCollisions()
	if g.Player.Stats.Damage <= oldDmg {
		t.Error("picking up Iron Tip should increase damage")
	}
}

func TestItemPickupMaxHPIncrease(t *testing.T) {
	g := testGame(ClassMage)
	item := findItem("Heart Container") // +2 HP
	oldHP := g.Player.HP
	oldMax := g.Player.MaxHP
	g.ItemDrops = append(g.ItemDrops, ItemDrop{
		X: g.Player.X, Z: g.Player.Z, Item: item,
	})
	g.checkCollisions()
	if g.Player.MaxHP != oldMax+2 {
		t.Errorf("max HP: want %d, got %d", oldMax+2, g.Player.MaxHP)
	}
	// HP should also increase by the diff
	if g.Player.HP != oldHP+2 {
		t.Errorf("current HP should increase with max: want %d, got %d", oldHP+2, g.Player.HP)
	}
}

// ---------- tickFloats ----------

func TestTickFloatsRemovesExpired(t *testing.T) {
	g := testGame(ClassMage)
	g.Floats = append(g.Floats,
		FloatingText{Timer: 0.5, MaxTime: 1.0},
		FloatingText{Timer: 0.01, MaxTime: 1.0},
	)
	g.tickFloats(0.1)
	if len(g.Floats) != 1 {
		t.Errorf("should have 1 float left, got %d", len(g.Floats))
	}
}

// ---------- tickParticles ----------

func TestTickParticlesGravity(t *testing.T) {
	g := testGame(ClassMage)
	g.Particles = append(g.Particles, Particle{
		Y: 5, VY: 0, Life: 1.0, MaxLife: 1.0,
	})
	g.tickParticles(0.1)
	if g.Particles[0].VY >= 0 {
		t.Error("gravity should pull particle down")
	}
}

func TestTickParticlesRemovesExpired(t *testing.T) {
	g := testGame(ClassMage)
	g.Particles = append(g.Particles,
		Particle{Life: 0.5, MaxLife: 1.0},
		Particle{Life: 0.01, MaxLife: 1.0},
	)
	g.tickParticles(0.1)
	if len(g.Particles) != 1 {
		t.Errorf("should have 1 particle left, got %d", len(g.Particles))
	}
}
