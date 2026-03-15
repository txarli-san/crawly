package main

import (
	"fmt"
	"math"
	"math/rand"
)

// ---------- Enemy types ----------

type EnemyType int

const (
	EnemyMinion  EnemyType = iota // melee, low HP, fast
	EnemyWarrior                  // melee, tanky, slow
	EnemyMage                     // ranged, squishy
)

// ---------- Core entities ----------

type Player struct {
	X, Z          float32
	FacingAngle   float32
	HP, MaxHP     int
	Items         []*ItemDef
	Stats         PlayerStats
	FireTimer     float32
	DodgeTimer    float32 // >0 = currently dodging
	DodgeCooldown float32 // >0 = can't dodge yet
	InvulnTimer   float32 // >0 = invulnerable
	DodgeVX       float32
	DodgeVZ       float32
	Moving        bool
	Anim          AnimState
	VisibleMeshes []bool
}

type Enemy struct {
	X, Z         float32
	FacingAngle  float32
	HP, MaxHP    int
	Type         EnemyType
	Speed        float32
	Damage       int
	AttackTimer  float32
	AttackRange  float32
	AttackRate   float32 // seconds between attacks
	Alive        bool
	Moving       bool
	Anim         AnimState

	// Status effects
	FireTimer   float32
	FireDamage  float32
	IceTimer    float32
	PoisonTimer float32
	PoisonDmg   float32
}

type Projectile struct {
	X, Z       float32
	VX, VZ     float32
	Damage     float32
	Radius     float32
	Alive      bool
	Owner      int // 0=player, 1=enemy
	Age        float32

	// Tags from items
	Fire      bool
	Ice       bool
	Poison    bool
	Pierce    bool
	PierceLeft int
	Bounce    bool
	BounceLeft int
	Homing    bool
	Explosive bool
	Chain     bool
	ChainLeft int
	Giant     bool
}

type ItemDrop struct {
	X, Z      float32
	Item      *ItemDef
	Timer     float32 // animation timer
	Collected bool
}

type Explosion struct {
	X, Z    float32
	Radius  float32
	Damage  float32
	Timer   float32
	MaxTime float32
	Owner   int
	Hit     map[int]bool // enemies already damaged
}

type FloatingText struct {
	Text     string
	WorldX   float32
	WorldZ   float32
	Timer    float32
	MaxTime  float32
	Color    [4]uint8
	FontSize int32
}

// ---------- Game state ----------

type GamePhase int

const (
	PhaseTitle      GamePhase = iota
	PhasePlaying
	PhaseTransition
	PhaseDead
)

type GameState struct {
	Phase       GamePhase
	Player      Player
	Enemies     []Enemy
	Projectiles []Projectile
	ItemDrops   []ItemDrop
	Explosions  []Explosion
	Floats      []FloatingText

	CurrentRoom *RoomDef
	RoomCoord   RoomCoord
	RoomMap     map[RoomCoord]*RoomDef

	Seed         int64
	Rng          *rand.Rand
	Score        int
	RoomsCleared int

	TransitionTimer float32
	TransitionDir   DoorDir
	DeathTimer      float32

	Message      string
	MessageTimer float32

	TileUnit      float32
	FloorSurfaceY float32
}

func NewGame(seed int64, tileUnit, floorSurfaceY float32) *GameState {
	rng := rand.New(rand.NewSource(seed))
	g := &GameState{
		Phase:         PhasePlaying,
		Seed:          seed,
		Rng:           rng,
		RoomMap:       make(map[RoomCoord]*RoomDef),
		TileUnit:      tileUnit,
		FloorSurfaceY: floorSurfaceY,
	}

	// Generate start room
	startCoord := RoomCoord{0, 0}
	var noDoors [DoorCount]bool
	startRoom := GenerateRoom(rng, startCoord, 0, noDoors)
	g.RoomMap[startCoord] = startRoom
	g.CurrentRoom = startRoom
	g.RoomCoord = startCoord

	// Place player at center
	g.Player = Player{
		X:     float32(RoomW) / 2.0 * tileUnit,
		Z:     float32(RoomH) / 2.0 * tileUnit,
		HP:    BaseHP,
		MaxHP: BaseHP,
		Stats: ComputeStats(nil),
	}
	g.Player.VisibleMeshes = playerVisibleMeshes()

	return g
}

func playerVisibleMeshes() []bool {
	// Mage: body meshes (6-11) + wand (2) + cape (5)
	v := make([]bool, 12)
	for i := 6; i <= 11; i++ {
		v[i] = true
	}
	v[2] = true // 1H_Wand
	v[5] = true // Cape
	return v
}

func (g *GameState) SetMessage(msg string) {
	g.Message = msg
	g.MessageTimer = 2.5
}

func (g *GameState) AddFloat(text string, wx, wz float32, r, gr, b uint8, size int32) {
	g.Floats = append(g.Floats, FloatingText{
		Text: text, WorldX: wx, WorldZ: wz,
		Timer: 1.2, MaxTime: 1.2,
		Color: [4]uint8{r, gr, b, 255}, FontSize: size,
	})
}

// ---------- Update ----------

const (
	dodgeDuration     = 0.2
	dodgeSpeed        = 18.0
	dodgeCooldownTime = 0.6
	colliderRadius    = 0.35
	pickupRadius      = 0.6
	projectileMaxAge  = 4.0
	enemyContactDmg   = 1
)

func (g *GameState) Update(dt float32) {
	switch g.Phase {
	case PhasePlaying:
		g.updatePlaying(dt)
	case PhaseTransition:
		g.TransitionTimer -= dt
		if g.TransitionTimer <= 0 {
			g.finishTransition()
		}
	case PhaseDead:
		g.DeathTimer -= dt
		g.tickFloats(dt)
	}
}

func (g *GameState) updatePlaying(dt float32) {
	g.updatePlayer(dt)
	g.updateEnemies(dt)
	g.updateProjectiles(dt)
	g.updateExplosions(dt)
	g.checkCollisions()
	g.checkRoomCleared()
	g.checkDoorTransition()
	g.tickFloats(dt)
	g.tickDrops(dt)

	if g.MessageTimer > 0 {
		g.MessageTimer -= dt
	}

	// Check death
	if g.Player.HP <= 0 {
		g.Phase = PhaseDead
		g.DeathTimer = 1.5
	}
}

func (g *GameState) updatePlayer(dt float32) {
	p := &g.Player

	// Dodge
	if p.DodgeTimer > 0 {
		p.DodgeTimer -= dt
		p.X += p.DodgeVX * dt
		p.Z += p.DodgeVZ * dt
		p.X, p.Z = g.resolveWallCollision(p.X, p.Z, colliderRadius)
		p.Moving = true
		return
	}
	if p.DodgeCooldown > 0 {
		p.DodgeCooldown -= dt
	}
	if p.InvulnTimer > 0 {
		p.InvulnTimer -= dt
	}

	// Movement input
	var dx, dz float32
	if isKeyDown(keyW) {
		dz -= 1
	}
	if isKeyDown(keyS) {
		dz += 1
	}
	if isKeyDown(keyA) {
		dx -= 1
	}
	if isKeyDown(keyD) {
		dx += 1
	}

	if dx != 0 || dz != 0 {
		length := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		dx /= length
		dz /= length
		p.X += dx * p.Stats.Speed * g.TileUnit * dt
		p.Z += dz * p.Stats.Speed * g.TileUnit * dt
		p.X, p.Z = g.resolveWallCollision(p.X, p.Z, colliderRadius*g.TileUnit)
		p.Moving = true
	} else {
		p.Moving = false
	}

	// Fire timer
	if p.FireTimer > 0 {
		p.FireTimer -= dt
	}
}

func (g *GameState) updateEnemies(dt float32) {
	tu := g.TileUnit
	p := &g.Player

	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}

		// Status effect ticks
		if e.FireTimer > 0 {
			e.FireTimer -= dt
			e.HP -= int(math.Ceil(float64(e.FireDamage * dt)))
			if e.HP <= 0 {
				g.killEnemy(i)
				continue
			}
		}
		if e.PoisonTimer > 0 {
			e.PoisonTimer -= dt
			e.HP -= int(math.Ceil(float64(e.PoisonDmg * dt)))
			if e.HP <= 0 {
				g.killEnemy(i)
				continue
			}
		}
		if e.IceTimer > 0 {
			e.IceTimer -= dt
		}

		// Slow factor from ice
		speedMult := float32(1.0)
		if e.IceTimer > 0 {
			speedMult = 0.35
		}

		// AI
		ddx := p.X - e.X
		ddz := p.Z - e.Z
		dist := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
		e.AttackTimer -= dt

		if dist > 0.1*tu {
			ndx := ddx / dist
			ndz := ddz / dist
			e.FacingAngle = float32(math.Atan2(float64(ndx), float64(ndz))) * 180 / math.Pi

			switch e.Type {
			case EnemyMinion, EnemyWarrior:
				// Melee: always chase into the player
				e.X += ndx * e.Speed * tu * speedMult * dt
				e.Z += ndz * e.Speed * tu * speedMult * dt
				e.X, e.Z = g.resolveWallCollision(e.X, e.Z, colliderRadius*tu)
				e.Moving = true

				// Melee attack on contact
				contactDist := colliderRadius * tu * 2.2
				if dist < contactDist && e.AttackTimer <= 0 {
					e.AttackTimer = e.AttackRate
					g.damagePlayer(e.Damage)
					// Knockback: push player away
					p.X += ndx * tu * 0.4
					p.Z += ndz * tu * 0.4
					p.X, p.Z = g.resolveWallCollision(p.X, p.Z, colliderRadius*tu)
				}

			case EnemyMage:
				// Ranged: keep distance, strafe, shoot
				preferDist := float32(4.0) * tu
				if dist < preferDist*0.7 {
					// Too close — flee
					e.X -= ndx * e.Speed * tu * speedMult * dt
					e.Z -= ndz * e.Speed * tu * speedMult * dt
					e.Moving = true
				} else if dist > preferDist*1.3 {
					// Too far — approach
					e.X += ndx * e.Speed * tu * speedMult * dt
					e.Z += ndz * e.Speed * tu * speedMult * dt
					e.Moving = true
				} else {
					// Good range — strafe perpendicular
					perpX := -ndz
					perpZ := ndx
					// Alternate strafe direction based on index
					if i%2 == 0 {
						perpX = ndz
						perpZ = -ndx
					}
					e.X += perpX * e.Speed * tu * speedMult * 0.6 * dt
					e.Z += perpZ * e.Speed * tu * speedMult * 0.6 * dt
					e.Moving = true
				}
				e.X, e.Z = g.resolveWallCollision(e.X, e.Z, colliderRadius*tu)

				// Shoot projectile
				if e.AttackTimer <= 0 && dist < e.AttackRange*tu {
					e.AttackTimer = e.AttackRate
					speed := float32(8.0) * tu
					g.Projectiles = append(g.Projectiles, Projectile{
						X: e.X, Z: e.Z,
						VX: ndx * speed, VZ: ndz * speed,
						Damage: float32(e.Damage), Radius: 0.1 * tu,
						Alive: true, Owner: 1,
					})
				}
			}
		}

		// Enemy-enemy separation
		for j := range g.Enemies {
			if i == j || !g.Enemies[j].Alive {
				continue
			}
			ox := e.X - g.Enemies[j].X
			oz := e.Z - g.Enemies[j].Z
			odist := float32(math.Sqrt(float64(ox*ox + oz*oz)))
			minDist := colliderRadius * tu * 1.8
			if odist < minDist && odist > 0.01 {
				push := (minDist - odist) * 0.5
				e.X += (ox / odist) * push
				e.Z += (oz / odist) * push
			}
		}
	}
}

func (g *GameState) updateProjectiles(dt float32) {
	tu := g.TileUnit

	for i := range g.Projectiles {
		proj := &g.Projectiles[i]
		if !proj.Alive {
			continue
		}

		proj.Age += dt
		if proj.Age > projectileMaxAge {
			proj.Alive = false
			continue
		}

		// Homing
		if proj.Homing && proj.Owner == 0 {
			var nearDist float32 = 999999
			var nearDX, nearDZ float32
			for j := range g.Enemies {
				e := &g.Enemies[j]
				if !e.Alive {
					continue
				}
				edx := e.X - proj.X
				edz := e.Z - proj.Z
				ed := float32(math.Sqrt(float64(edx*edx + edz*edz)))
				if ed < nearDist {
					nearDist = ed
					nearDX = edx / ed
					nearDZ = edz / ed
				}
			}
			if nearDist < 8*tu {
				homingStr := float32(3.0) * dt
				proj.VX += nearDX * homingStr * tu
				proj.VZ += nearDZ * homingStr * tu
				// Re-normalize speed
				speed := float32(math.Sqrt(float64(proj.VX*proj.VX + proj.VZ*proj.VZ)))
				targetSpeed := g.Player.Stats.ProjSpeed * tu
				if speed > 0.01 {
					proj.VX = proj.VX / speed * targetSpeed
					proj.VZ = proj.VZ / speed * targetSpeed
				}
			}
		}

		proj.X += proj.VX * dt
		proj.Z += proj.VZ * dt

		// Wall collision
		tx := int(math.Floor(float64(proj.X / tu)))
		tz := int(math.Floor(float64(proj.Z / tu)))
		if tx < 0 || tx >= RoomW || tz < 0 || tz >= RoomH || !g.isTileWalkable(tx, tz) {
			if proj.Bounce && proj.BounceLeft > 0 {
				proj.BounceLeft--
				// Reflect off wall — find which axis
				prevTX := int(math.Floor(float64((proj.X - proj.VX*dt) / tu)))
				prevTZ := int(math.Floor(float64((proj.Z - proj.VZ*dt) / tu)))
				if prevTX != tx {
					proj.VX = -proj.VX
				}
				if prevTZ != tz {
					proj.VZ = -proj.VZ
				}
				proj.X += proj.VX * dt * 2
				proj.Z += proj.VZ * dt * 2
			} else {
				proj.Alive = false
				if proj.Explosive {
					g.spawnExplosion(proj.X, proj.Z, proj.Damage, proj.Owner)
				}
			}
		}
	}
}

func (g *GameState) updateExplosions(dt float32) {
	tu := g.TileUnit
	for i := range g.Explosions {
		exp := &g.Explosions[i]
		if exp.Timer <= 0 {
			continue
		}
		exp.Timer -= dt

		// Damage enemies in radius (once per enemy)
		if exp.Owner == 0 {
			for j := range g.Enemies {
				e := &g.Enemies[j]
				if !e.Alive {
					continue
				}
				if exp.Hit[j] {
					continue
				}
				ddx := e.X - exp.X
				ddz := e.Z - exp.Z
				dist := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
				if dist < exp.Radius*tu {
					e.HP -= int(exp.Damage)
					exp.Hit[j] = true
					g.AddFloat(fmt.Sprintf("-%d", int(exp.Damage)), e.X, e.Z, 255, 180, 40, 18)
					if e.HP <= 0 {
						g.killEnemy(j)
					}
				}
			}
		}
	}
}

func (g *GameState) checkCollisions() {
	tu := g.TileUnit
	p := &g.Player

	// Player projectiles vs enemies
	for i := range g.Projectiles {
		proj := &g.Projectiles[i]
		if !proj.Alive || proj.Owner != 0 {
			continue
		}
		for j := range g.Enemies {
			e := &g.Enemies[j]
			if !e.Alive {
				continue
			}
			ddx := e.X - proj.X
			ddz := e.Z - proj.Z
			dist := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
			if dist < (proj.Radius+colliderRadius*tu) {
				// Hit!
				dmg := int(proj.Damage)
				e.HP -= dmg
				g.AddFloat(fmt.Sprintf("-%d", dmg), e.X, e.Z, 255, 255, 200, 16)

				// Apply status effects
				if proj.Fire {
					e.FireTimer = 3.0
					e.FireDamage = float32(g.Player.Stats.FireStacks) * 1.5
				}
				if proj.Ice {
					e.IceTimer = 2.0 + float32(g.Player.Stats.IceStacks)*0.5
				}
				if proj.Poison {
					e.PoisonTimer = 4.0
					e.PoisonDmg = float32(g.Player.Stats.PoisonStacks) * 1.0
				}

				if e.HP <= 0 {
					g.killEnemy(j)
					// Vampiric heal
					if g.Player.Stats.VampiricStacks > 0 {
						heal := g.Player.Stats.VampiricStacks
						p.HP += heal
						if p.HP > p.MaxHP {
							p.HP = p.MaxHP
						}
						g.AddFloat(fmt.Sprintf("+%d", heal), p.X, p.Z, 80, 255, 80, 16)
					}
				}

				// Chain to nearby enemy
				if proj.Chain && proj.ChainLeft > 0 {
					proj.ChainLeft--
					var nearestIdx = -1
					var nearestDist float32 = 999999
					for k := range g.Enemies {
						if k == j || !g.Enemies[k].Alive {
							continue
						}
						cdx := g.Enemies[k].X - e.X
						cdz := g.Enemies[k].Z - e.Z
						cd := float32(math.Sqrt(float64(cdx*cdx + cdz*cdz)))
						if cd < nearestDist && cd < 5*tu {
							nearestDist = cd
							nearestIdx = k
						}
					}
					if nearestIdx >= 0 {
						// Redirect projectile toward chained enemy
						target := &g.Enemies[nearestIdx]
						cdx := target.X - proj.X
						cdz := target.Z - proj.Z
						cd := float32(math.Sqrt(float64(cdx*cdx + cdz*cdz)))
						if cd > 0.01 {
							speed := float32(math.Sqrt(float64(proj.VX*proj.VX + proj.VZ*proj.VZ)))
							proj.VX = (cdx / cd) * speed
							proj.VZ = (cdz / cd) * speed
						}
						continue // Don't kill the projectile
					}
				}

				// Pierce
				if proj.Pierce && proj.PierceLeft > 0 {
					proj.PierceLeft--
					continue // Don't kill the projectile
				}

				// Explosive
				if proj.Explosive {
					g.spawnExplosion(proj.X, proj.Z, proj.Damage*0.6, proj.Owner)
				}

				proj.Alive = false
				break
			}
		}
	}

	// Enemy projectiles vs player
	for i := range g.Projectiles {
		proj := &g.Projectiles[i]
		if !proj.Alive || proj.Owner != 1 {
			continue
		}
		ddx := p.X - proj.X
		ddz := p.Z - proj.Z
		dist := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
		if dist < (proj.Radius + colliderRadius*tu) {
			g.damagePlayer(int(proj.Damage))
			proj.Alive = false
		}
	}

	// Player vs item drops
	for i := range g.ItemDrops {
		drop := &g.ItemDrops[i]
		if drop.Collected {
			continue
		}
		ddx := p.X - drop.X
		ddz := p.Z - drop.Z
		dist := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
		if dist < pickupRadius*tu {
			drop.Collected = true
			p.Items = append(p.Items, drop.Item)
			p.Stats = ComputeStats(p.Items)

			// Apply max HP increase
			if p.MaxHP < p.Stats.MaxHP {
				diff := p.Stats.MaxHP - p.MaxHP
				p.MaxHP = p.Stats.MaxHP
				p.HP += diff
			}

			// Reset shields per room
			g.SetMessage(drop.Item.Name + ": " + drop.Item.Desc)
			g.AddFloat(drop.Item.Name, drop.X, drop.Z, 255, 220, 100, 18)
		}
	}
}

func (g *GameState) damagePlayer(dmg int) {
	p := &g.Player
	if p.InvulnTimer > 0 || p.DodgeTimer > 0 {
		return
	}
	// Shield absorb
	if p.Stats.ShieldStacks > 0 {
		p.Stats.ShieldStacks--
		g.AddFloat("BLOCKED", p.X, p.Z, 100, 200, 255, 18)
		p.InvulnTimer = 0.3
		return
	}
	p.HP -= dmg
	p.InvulnTimer = 0.8
	g.AddFloat(fmt.Sprintf("-%d", dmg), p.X, p.Z, 255, 60, 60, 20)
}

func (g *GameState) killEnemy(idx int) {
	e := &g.Enemies[idx]
	e.Alive = false
	g.Score += 10 * (int(e.Type) + 1)
	g.AddFloat("SLAIN", e.X, e.Z, 255, 200, 60, 16)

	// Item drop chance
	dropChance := 0.30
	if g.Rng.Float64() < dropChance {
		item := RollItemDrop(g.Rng, g.CurrentRoom.Depth)
		g.ItemDrops = append(g.ItemDrops, ItemDrop{
			X: e.X, Z: e.Z, Item: item, Timer: 0,
		})
	}
}

func (g *GameState) spawnExplosion(x, z, damage float32, owner int) {
	radius := float32(1.5) + float32(g.Player.Stats.ExplosiveStacks)*0.5
	g.Explosions = append(g.Explosions, Explosion{
		X: x, Z: z, Radius: radius, Damage: damage,
		Timer: 0.3, MaxTime: 0.3, Owner: owner,
		Hit: make(map[int]bool),
	})
}

func (g *GameState) checkRoomCleared() {
	room := g.CurrentRoom
	if room == nil || room.Cleared {
		return
	}
	for i := range g.Enemies {
		if g.Enemies[i].Alive {
			return
		}
	}
	// All enemies dead — open doors
	room.Cleared = true
	g.RoomsCleared++
	g.SetMessage("Room Cleared!")

	// Reset shields for new room
	g.Player.Stats.ShieldStacks = ComputeStats(g.Player.Items).ShieldStacks

	for z := 0; z < RoomH; z++ {
		for x := 0; x < RoomW; x++ {
			if room.Tiles[z][x] == TileDoorClosed {
				room.Tiles[z][x] = TileDoorOpen
			}
		}
	}
}

func (g *GameState) checkDoorTransition() {
	tu := g.TileUnit
	p := &g.Player
	room := g.CurrentRoom
	if room == nil || !room.Cleared {
		return
	}

	// Check if player is standing on a door tile
	tx := int(math.Floor(float64(p.X / tu)))
	tz := int(math.Floor(float64(p.Z / tu)))

	for d := DoorDir(0); d < DoorCount; d++ {
		if !room.Doors[d] {
			continue
		}
		tiles := doorTiles(d)
		for _, dt := range tiles {
			if tx == dt[0] && tz == dt[1] {
				// Check player is at the edge (actually stepping through)
				atEdge := false
				switch d {
				case DoorNorth:
					atEdge = p.Z < tu*1.2
				case DoorSouth:
					atEdge = p.Z > float32(RoomH-1)*tu-tu*0.2
				case DoorWest:
					atEdge = p.X < tu*1.2
				case DoorEast:
					atEdge = p.X > float32(RoomW-1)*tu-tu*0.2
				}
				if atEdge {
					g.beginTransition(d)
					return
				}
			}
		}
	}
}

func (g *GameState) beginTransition(dir DoorDir) {
	g.Phase = PhaseTransition
	g.TransitionTimer = 0.5
	g.TransitionDir = dir
}

func (g *GameState) finishTransition() {
	dir := g.TransitionDir
	newCoord := NeighborCoord(g.RoomCoord, dir)

	// Get or create the room
	room, exists := g.RoomMap[newCoord]
	if !exists {
		// Determine required doors (the one we're coming from)
		var required [DoorCount]bool
		required[dir.Opposite()] = true
		depth := g.RoomsCleared
		room = GenerateRoom(g.Rng, newCoord, depth, required)
		g.RoomMap[newCoord] = room
	}

	g.CurrentRoom = room
	g.RoomCoord = newCoord

	// Spawn enemies
	g.Enemies = g.Enemies[:0]
	for _, spawn := range room.Spawns {
		g.Enemies = append(g.Enemies, makeEnemy(spawn, g.TileUnit, room.Depth))
	}

	// Clear projectiles and drops from previous room
	g.Projectiles = g.Projectiles[:0]
	g.ItemDrops = g.ItemDrops[:0]
	g.Explosions = g.Explosions[:0]

	// Position player at the entry door
	entryX, entryZ := SpawnPos(dir.Opposite())
	g.Player.X = entryX * g.TileUnit
	g.Player.Z = entryZ * g.TileUnit

	// Reset shields for new room
	g.Player.Stats.ShieldStacks = ComputeStats(g.Player.Items).ShieldStacks

	g.Phase = PhasePlaying
}

func makeEnemy(spawn EnemySpawn, tileUnit float32, depth int) Enemy {
	e := Enemy{
		X:     spawn.X * tileUnit,
		Z:     spawn.Z * tileUnit,
		Type:  spawn.Type,
		Alive: true,
	}

	// Scale stats with depth
	hpScale := 1.0 + float64(depth)*0.12

	switch spawn.Type {
	case EnemyMinion:
		e.HP = int(float64(6) * hpScale)
		e.MaxHP = e.HP
		e.Speed = 3.5
		e.Damage = 1
		e.AttackRange = 1.2
		e.AttackRate = 0.8
	case EnemyWarrior:
		e.HP = int(float64(12) * hpScale)
		e.MaxHP = e.HP
		e.Speed = 2.2
		e.Damage = 2
		e.AttackRange = 1.5
		e.AttackRate = 1.2
	case EnemyMage:
		e.HP = int(float64(5) * hpScale)
		e.MaxHP = e.HP
		e.Speed = 2.0
		e.Damage = 2
		e.AttackRange = 6.0
		e.AttackRate = 1.8
	}

	return e
}

// SpawnPlayerProjectiles creates projectiles based on current item stats.
func (g *GameState) SpawnPlayerProjectiles(aimDX, aimDZ float32) {
	p := &g.Player
	tu := g.TileUnit
	stats := &p.Stats
	speed := stats.ProjSpeed * tu

	count := stats.ProjCount
	baseAngle := float32(math.Atan2(float64(aimDX), float64(aimDZ)))

	for i := 0; i < count; i++ {
		// Spread projectiles in a fan
		spread := float32(0)
		if count > 1 {
			spread = (float32(i) - float32(count-1)/2.0) * 0.15
		}
		angle := baseAngle + spread
		vx := float32(math.Sin(float64(angle))) * speed
		vz := float32(math.Cos(float64(angle))) * speed

		proj := Projectile{
			X: p.X, Z: p.Z,
			VX: vx, VZ: vz,
			Damage:     stats.Damage,
			Radius:     stats.ProjSize * tu,
			Alive:      true,
			Owner:      0,
			Fire:       stats.FireStacks > 0,
			Ice:        stats.IceStacks > 0,
			Poison:     stats.PoisonStacks > 0,
			Pierce:     stats.PierceCount > 0,
			PierceLeft: stats.PierceCount,
			Bounce:     stats.BounceCount > 0,
			BounceLeft: stats.BounceCount,
			Homing:     stats.HomingStacks > 0,
			Explosive:  stats.ExplosiveStacks > 0,
			Chain:      stats.ChainStacks > 0,
			ChainLeft:  stats.ChainStacks,
			Giant:      stats.GiantStacks > 0,
		}
		g.Projectiles = append(g.Projectiles, proj)
	}
}

// ---------- Collision helpers ----------

func (g *GameState) isTileWalkable(tx, tz int) bool {
	if tx < 0 || tx >= RoomW || tz < 0 || tz >= RoomH {
		return false
	}
	if g.CurrentRoom == nil {
		return false
	}
	t := g.CurrentRoom.Tiles[tz][tx]
	return t == TileFloor || t == TileDoorOpen
}

func (g *GameState) resolveWallCollision(x, z, radius float32) (float32, float32) {
	tu := g.TileUnit

	// Check the 4 tiles the entity might overlap with
	checkX := []float32{x - radius, x + radius}
	checkZ := []float32{z - radius, z + radius}

	for _, cx := range checkX {
		for _, cz := range checkZ {
			tx := int(math.Floor(float64(cx / tu)))
			tz := int(math.Floor(float64(cz / tu)))
			if !g.isTileWalkable(tx, tz) {
				// Push out
				tileCX := (float32(tx) + 0.5) * tu
				tileCZ := (float32(tz) + 0.5) * tu
				halfTile := tu * 0.5

				// Find overlap on each axis
				overlapX := (halfTile + radius) - float32(math.Abs(float64(x-tileCX)))
				overlapZ := (halfTile + radius) - float32(math.Abs(float64(z-tileCZ)))

				if overlapX > 0 && overlapZ > 0 {
					// Push on smallest axis
					if overlapX < overlapZ {
						if x < tileCX {
							x -= overlapX
						} else {
							x += overlapX
						}
					} else {
						if z < tileCZ {
							z -= overlapZ
						} else {
							z += overlapZ
						}
					}
				}
			}
		}
	}

	return x, z
}

// ---------- Float/misc helpers ----------

func (g *GameState) tickFloats(dt float32) {
	alive := g.Floats[:0]
	for i := range g.Floats {
		g.Floats[i].Timer -= dt
		if g.Floats[i].Timer > 0 {
			alive = append(alive, g.Floats[i])
		}
	}
	g.Floats = alive
}

func (g *GameState) tickDrops(dt float32) {
	for i := range g.ItemDrops {
		g.ItemDrops[i].Timer += dt
	}
}

