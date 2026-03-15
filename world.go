package main

import (
	"math/rand"
)

const (
	RoomW = 15 // total room width in tiles (including walls)
	RoomH = 11 // total room height in tiles (including walls)
)

type TileType int

const (
	TileFloor      TileType = iota // walkable
	TileWall                       // impassable wall
	TileDoorClosed                 // locked door (becomes open when room cleared)
	TileDoorOpen                   // open door (walkable, triggers room transition)
	TilePillar                     // obstacle pillar (blocks movement)
)

type DoorDir int

const (
	DoorNorth DoorDir = iota
	DoorSouth
	DoorEast
	DoorWest
	DoorCount
)

// Opposite returns the door on the other side when entering a room.
func (d DoorDir) Opposite() DoorDir {
	switch d {
	case DoorNorth:
		return DoorSouth
	case DoorSouth:
		return DoorNorth
	case DoorEast:
		return DoorWest
	case DoorWest:
		return DoorEast
	}
	return DoorNorth
}

type EnemySpawn struct {
	X, Z  float32
	Type  EnemyType
	Alert bool // spawns in alert state (facing door)
}

// Prop IDs — indexes into PropModels.Models
const (
	PropBarrel = iota
	PropBarrelDark
	PropCrate
	PropCrateDark
	PropBucket
	PropPots
	PropWeaponRack
	PropBench
	PropTableMedium
	PropStool
	PropBanner
	PropBookcaseFilled
	PropBookcaseWideFilled
	PropTableSmall
	PropBookA
	PropBookOpenA
	PropSpellBook
	PropTableLarge
	PropChair
	PropMug
	PropPlate
	PropPlateFull
	PropPillar
	PropPillarBroken
	PropBricks
	PropFloorDecoShattered
	PropTileSpikes
	PropTileSpikesLarge
	PropTorchWall
	PropFloorDecoTiles
	PropCount
)

type Prop struct {
	Model    int
	Rotation float32
	Wall     int  // -1=center, 0=N, 1=S, 2=W, 3=E
	Blocking bool
}

type RoomDef struct {
	Tiles   [RoomH][RoomW]TileType
	Doors   [DoorCount]bool
	Spawns  []EnemySpawn
	Props   map[[2]int]Prop
	Tags    TagSet
	Depth   int
	Cleared bool
}

type RoomCoord struct {
	X, Z int
}

// doorPos returns the tile positions for a 3-wide door on the given wall.
func doorTiles(dir DoorDir) [][2]int {
	mid := 0
	switch dir {
	case DoorNorth, DoorSouth:
		mid = RoomW / 2
		row := 0
		if dir == DoorSouth {
			row = RoomH - 1
		}
		return [][2]int{{mid - 1, row}, {mid, row}, {mid + 1, row}}
	case DoorWest, DoorEast:
		mid = RoomH / 2
		col := 0
		if dir == DoorEast {
			col = RoomW - 1
		}
		return [][2]int{{col, mid - 1}, {col, mid}, {col, mid + 1}}
	}
	return nil
}

// GenerateRoom creates a procedural room. `requiredDoors` indicates which doors
// must exist (because adjacent rooms already have connecting doors).
func GenerateRoom(rng *rand.Rand, coord RoomCoord, depth int, requiredDoors [DoorCount]bool, playerTags *TagSet) *RoomDef {
	room := &RoomDef{Depth: depth, Tags: NewTagSet()}

	// Fill with walls
	for z := 0; z < RoomH; z++ {
		for x := 0; x < RoomW; x++ {
			room.Tiles[z][x] = TileWall
		}
	}

	// Carve interior floor (1-tile border = walls)
	for z := 1; z < RoomH-1; z++ {
		for x := 1; x < RoomW-1; x++ {
			room.Tiles[z][x] = TileFloor
		}
	}

	// Determine doors: required ones + random extras
	for d := DoorDir(0); d < DoorCount; d++ {
		if requiredDoors[d] {
			room.Doors[d] = true
		}
	}
	// Add 1-3 random doors beyond required
	extraDoors := 1 + rng.Intn(3)
	for i := 0; i < extraDoors; i++ {
		d := DoorDir(rng.Intn(int(DoorCount)))
		room.Doors[d] = true
	}
	// Start room always has all doors
	if coord.X == 0 && coord.Z == 0 {
		for d := DoorDir(0); d < DoorCount; d++ {
			room.Doors[d] = true
		}
	}

	// Place doors as closed tiles (opened when room is cleared)
	isStartRoom := coord.X == 0 && coord.Z == 0
	for d := DoorDir(0); d < DoorCount; d++ {
		if !room.Doors[d] {
			continue
		}
		tiles := doorTiles(d)
		for _, t := range tiles {
			if isStartRoom {
				room.Tiles[t[1]][t[0]] = TileDoorOpen
			} else {
				room.Tiles[t[1]][t[0]] = TileDoorClosed
			}
		}
	}

	// Place interior obstacles and props
	room.Props = make(map[[2]int]Prop)
	if !isStartRoom {
		placeObstacles(rng, room, depth)
		placeEnemies(rng, room, depth, playerTags)
	}
	placeProps(rng, room)

	if isStartRoom {
		room.Cleared = true
	}

	return room
}

func placeObstacles(rng *rand.Rand, room *RoomDef, depth int) {
	// Number of pillars scales with depth (tuned for smaller rooms)
	numPillars := 1 + rng.Intn(2) + depth/4
	if numPillars > 6 {
		numPillars = 6
	}

	for i := 0; i < numPillars; i++ {
		for attempt := 0; attempt < 20; attempt++ {
			x := 2 + rng.Intn(RoomW-4)
			z := 2 + rng.Intn(RoomH-4)
			if room.Tiles[z][x] != TileFloor {
				continue
			}
			// Don't block doors — stay away from door areas
			blocked := false
			for d := DoorDir(0); d < DoorCount; d++ {
				if !room.Doors[d] {
					continue
				}
				tiles := doorTiles(d)
				for _, t := range tiles {
					dx, dz := x-t[0], z-t[1]
					if dx*dx+dz*dz < 9 {
						blocked = true
						break
					}
				}
				if blocked {
					break
				}
			}
			if blocked {
				continue
			}
			room.Tiles[z][x] = TilePillar
			break
		}
	}

	// Occasional wall formations (L-shapes, crosses) at higher depths
	if depth >= 3 && rng.Intn(3) == 0 {
		cx := 4 + rng.Intn(RoomW-8)
		cz := 4 + rng.Intn(RoomH-8)
		// Small cross
		offsets := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, off := range offsets {
			tx, tz := cx+off[0], cz+off[1]
			if tx > 1 && tx < RoomW-2 && tz > 1 && tz < RoomH-2 && room.Tiles[tz][tx] == TileFloor {
				room.Tiles[tz][tx] = TilePillar
			}
		}
	}
}

func placeEnemies(rng *rand.Rand, room *RoomDef, depth int, playerTags *TagSet) {
	// Enemy count scales with depth (tuned for smaller rooms)
	baseCount := 2 + depth/3
	if baseCount > 5 {
		baseCount = 5
	}
	count := baseCount + rng.Intn(2)

	// Tag-biased count adjustments
	if playerTags != nil {
		if playerTags.Has(TagWounded) {
			// Dungeon senses weakness — fewer but tougher enemies
			count--
		}
		if playerTags.Has(TagBerserk) {
			// Dungeon responds to aggression — more enemies
			count++
		}
		if playerTags.Has(TagStealthy) {
			// Fewer enemies, possible ambush advantage
			count--
		}
	}
	if count < 1 {
		count = 1
	}

	// Tag-biased enemy type: player tags shift distribution
	alertSpawn := playerTags != nil && playerTags.Has(TagLoud)
	moreElites := playerTags != nil && (playerTags.Has(TagBleeding) || playerTags.Has(TagCursed))

	for i := 0; i < count; i++ {
		var etype EnemyType
		roll := rng.Float64()

		if moreElites {
			// More dangerous composition when player is vulnerable
			switch {
			case depth < 3:
				if roll < 0.5 {
					etype = EnemyMinion
				} else {
					etype = EnemyWarrior
				}
			default:
				if roll < 0.25 {
					etype = EnemyMinion
				} else if roll < 0.55 {
					etype = EnemyWarrior
				} else {
					etype = EnemyMage
				}
			}
		} else {
			switch {
			case depth < 3:
				etype = EnemyMinion
			case depth < 6:
				if roll < 0.6 {
					etype = EnemyMinion
				} else {
					etype = EnemyWarrior
				}
			default:
				if roll < 0.4 {
					etype = EnemyMinion
				} else if roll < 0.75 {
					etype = EnemyWarrior
				} else {
					etype = EnemyMage
				}
			}
		}

		// Find a valid spawn position
		for attempt := 0; attempt < 30; attempt++ {
			x := 3 + rng.Intn(RoomW-6)
			z := 3 + rng.Intn(RoomH-6)
			if room.Tiles[z][x] != TileFloor {
				continue
			}
			spawn := EnemySpawn{
				X:    float32(x) + 0.5,
				Z:    float32(z) + 0.5,
				Type: etype,
			}
			// If player is loud, spawn near doors (alert ambush)
			if alertSpawn && rng.Float64() < 0.5 {
				// Bias toward door positions
				for d := DoorDir(0); d < DoorCount; d++ {
					if !room.Doors[d] {
						continue
					}
					tiles := doorTiles(d)
					mid := tiles[1]
					spawn.X = float32(mid[0]) + 0.5
					spawn.Z = float32(mid[1]) + 0.5
					// Offset inward
					switch d {
					case DoorNorth:
						spawn.Z += 2
					case DoorSouth:
						spawn.Z -= 2
					case DoorWest:
						spawn.X += 2
					case DoorEast:
						spawn.X -= 2
					}
					spawn.Alert = true
					break
				}
			}
			room.Spawns = append(room.Spawns, spawn)
			break
		}
	}
}

// SpawnPos returns the world position (in tile-space units, multiply by tileUnit)
// where the player enters from a given door direction.
func SpawnPos(fromDoor DoorDir) (float32, float32) {
	switch fromDoor {
	case DoorNorth:
		return float32(RoomW) / 2.0, 2.0
	case DoorSouth:
		return float32(RoomW) / 2.0, float32(RoomH) - 2.0
	case DoorEast:
		return float32(RoomW) - 2.0, float32(RoomH) / 2.0
	case DoorWest:
		return 2.0, float32(RoomH) / 2.0
	}
	// Default: center
	return float32(RoomW) / 2.0, float32(RoomH) / 2.0
}

// NeighborCoord returns the room coordinate when going through a door.
func NeighborCoord(from RoomCoord, dir DoorDir) RoomCoord {
	switch dir {
	case DoorNorth:
		return RoomCoord{from.X, from.Z - 1}
	case DoorSouth:
		return RoomCoord{from.X, from.Z + 1}
	case DoorEast:
		return RoomCoord{from.X + 1, from.Z}
	case DoorWest:
		return RoomCoord{from.X - 1, from.Z}
	}
	return from
}

// --- Room themes and prop placement ---

type roomTheme int

const (
	themeDungeon roomTheme = iota
	themeStorage
	themeLibrary
	themeCrypt
)

type propPalette struct {
	wallProps    []int
	centerProps  []int
	wallChance   float64
	centerChance float64
}

var themePalettes = map[roomTheme]propPalette{
	themeDungeon: {
		wallProps:    []int{PropWeaponRack, PropBench, PropTorchWall, PropBanner},
		centerProps:  []int{PropStool, PropBricks, PropFloorDecoTiles},
		wallChance:   0.25,
		centerChance: 0.08,
	},
	themeStorage: {
		wallProps:    []int{PropBarrel, PropBarrelDark, PropCrate, PropCrateDark},
		centerProps:  []int{PropBucket, PropPots},
		wallChance:   0.40,
		centerChance: 0.12,
	},
	themeLibrary: {
		wallProps:    []int{PropBookcaseFilled, PropBookcaseWideFilled, PropTableSmall},
		centerProps:  []int{PropBookA, PropBookOpenA, PropSpellBook},
		wallChance:   0.35,
		centerChance: 0.12,
	},
	themeCrypt: {
		wallProps:    []int{PropPillar, PropPillarBroken, PropTorchWall},
		centerProps:  []int{PropBanner, PropFloorDecoShattered, PropFloorDecoTiles},
		wallChance:   0.30,
		centerChance: 0.10,
	},
}

func pickTheme(rng *rand.Rand) roomTheme {
	themes := []roomTheme{themeDungeon, themeStorage, themeLibrary, themeCrypt}
	return themes[rng.Intn(len(themes))]
}

func wallRotation(wallDir int) float32 {
	switch wallDir {
	case 0:
		return 180
	case 1:
		return 0
	case 2:
		return 90
	case 3:
		return -90
	}
	return 0
}

func placeProps(rng *rand.Rand, room *RoomDef) {
	theme := pickTheme(rng)
	pal := themePalettes[theme]

	// Collect which tiles have enemy spawns
	spawnTiles := map[[2]int]bool{}
	for _, s := range room.Spawns {
		spawnTiles[[2]int{int(s.X), int(s.Z)}] = true
	}

	// Collect door tiles to keep clear
	doorZone := map[[2]int]bool{}
	for d := DoorDir(0); d < DoorCount; d++ {
		if !room.Doors[d] {
			continue
		}
		tiles := doorTiles(d)
		for _, t := range tiles {
			// Mark door tile and neighbors as clear
			for dz := -1; dz <= 1; dz++ {
				for dx := -1; dx <= 1; dx++ {
					doorZone[[2]int{t[0] + dx, t[1] + dz}] = true
				}
			}
		}
	}

	for z := 1; z < RoomH-1; z++ {
		for x := 1; x < RoomW-1; x++ {
			if room.Tiles[z][x] != TileFloor {
				continue
			}
			key := [2]int{x, z}
			if spawnTiles[key] || doorZone[key] {
				continue
			}

			// Check if adjacent to a wall
			isWallAdj := false
			wallDir := -1
			dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
			for i, d := range dirs {
				nx, nz := x+d[0], z+d[1]
				if nx >= 0 && nx < RoomW && nz >= 0 && nz < RoomH {
					if room.Tiles[nz][nx] == TileWall {
						isWallAdj = true
						wallDir = i
						break
					}
				}
			}

			if isWallAdj && len(pal.wallProps) > 0 && rng.Float64() < pal.wallChance {
				model := pal.wallProps[rng.Intn(len(pal.wallProps))]
				room.Props[key] = Prop{
					Model:    model,
					Rotation: wallRotation(wallDir),
					Wall:     wallDir,
					Blocking: true,
				}
			} else if !isWallAdj && len(pal.centerProps) > 0 && rng.Float64() < pal.centerChance {
				model := pal.centerProps[rng.Intn(len(pal.centerProps))]
				room.Props[key] = Prop{
					Model:    model,
					Rotation: float32(rng.Intn(4)) * 90,
					Wall:     -1,
					Blocking: false,
				}
			}
		}
	}

	// Always place torches on a few wall-adjacent spots for atmosphere
	torchCount := 2 + rng.Intn(3)
	placed := 0
	for attempt := 0; attempt < 50 && placed < torchCount; attempt++ {
		x := 1 + rng.Intn(RoomW-2)
		z := 1 + rng.Intn(RoomH-2)
		if room.Tiles[z][x] != TileFloor {
			continue
		}
		key := [2]int{x, z}
		if _, exists := room.Props[key]; exists {
			continue
		}
		if doorZone[key] {
			continue
		}
		// Must be adjacent to wall
		for i, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			nx, nz := x+d[0], z+d[1]
			if nx >= 0 && nx < RoomW && nz >= 0 && nz < RoomH && room.Tiles[nz][nx] == TileWall {
				room.Props[key] = Prop{
					Model:    PropTorchWall,
					Rotation: wallRotation(i),
					Wall:     i,
					Blocking: false,
				}
				placed++
				break
			}
		}
	}
}
