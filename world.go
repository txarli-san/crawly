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
	X, Z float32
	Type EnemyType
}

type RoomDef struct {
	Tiles   [RoomH][RoomW]TileType
	Doors   [DoorCount]bool
	Spawns  []EnemySpawn
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
func GenerateRoom(rng *rand.Rand, coord RoomCoord, depth int, requiredDoors [DoorCount]bool) *RoomDef {
	room := &RoomDef{Depth: depth}

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

	// Place interior obstacles based on depth
	if !isStartRoom {
		placeObstacles(rng, room, depth)
		placeEnemies(rng, room, depth, coord)
	}

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

func placeEnemies(rng *rand.Rand, room *RoomDef, depth int, coord RoomCoord) {
	// Enemy count scales with depth (tuned for smaller rooms)
	baseCount := 2 + depth/3
	if baseCount > 5 {
		baseCount = 5
	}
	count := baseCount + rng.Intn(2)

	// Enemy type distribution shifts with depth
	for i := 0; i < count; i++ {
		var etype EnemyType
		roll := rng.Float64()
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

		// Find a valid spawn position (floor tile, away from doors)
		for attempt := 0; attempt < 30; attempt++ {
			x := 3 + rng.Intn(RoomW-6)
			z := 3 + rng.Intn(RoomH-6)
			if room.Tiles[z][x] != TileFloor {
				continue
			}
			room.Spawns = append(room.Spawns, EnemySpawn{
				X:    float32(x) + 0.5,
				Z:    float32(z) + 0.5,
				Type: etype,
			})
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
