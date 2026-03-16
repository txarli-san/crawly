package main

import (
	"math/rand"
	"testing"
)

// ---------- DoorDir.Opposite ----------

func TestDoorOppositeSymmetry(t *testing.T) {
	dirs := []DoorDir{DoorNorth, DoorSouth, DoorEast, DoorWest}
	for _, d := range dirs {
		if d.Opposite().Opposite() != d {
			t.Errorf("Opposite().Opposite() of %d != %d", d, d)
		}
	}
}

func TestDoorOppositeValues(t *testing.T) {
	cases := [][2]DoorDir{
		{DoorNorth, DoorSouth},
		{DoorSouth, DoorNorth},
		{DoorEast, DoorWest},
		{DoorWest, DoorEast},
	}
	for _, c := range cases {
		if c[0].Opposite() != c[1] {
			t.Errorf("Opposite of %d: want %d, got %d", c[0], c[1], c[0].Opposite())
		}
	}
}

// ---------- doorTiles ----------

func TestDoorTilesThreePerDoor(t *testing.T) {
	for d := DoorDir(0); d < DoorCount; d++ {
		tiles := doorTiles(d)
		if len(tiles) != 3 {
			t.Errorf("door %d: want 3 tiles, got %d", d, len(tiles))
		}
	}
}

func TestDoorTilesOnWallEdge(t *testing.T) {
	// North door tiles should be on row 0
	for _, tile := range doorTiles(DoorNorth) {
		if tile[1] != 0 {
			t.Errorf("north door tile z=%d, want 0", tile[1])
		}
	}
	// South door tiles on last row
	for _, tile := range doorTiles(DoorSouth) {
		if tile[1] != RoomH-1 {
			t.Errorf("south door tile z=%d, want %d", tile[1], RoomH-1)
		}
	}
	// West door tiles on column 0
	for _, tile := range doorTiles(DoorWest) {
		if tile[0] != 0 {
			t.Errorf("west door tile x=%d, want 0", tile[0])
		}
	}
	// East door tiles on last column
	for _, tile := range doorTiles(DoorEast) {
		if tile[0] != RoomW-1 {
			t.Errorf("east door tile x=%d, want %d", tile[0], RoomW-1)
		}
	}
}

func TestDoorTilesCentered(t *testing.T) {
	// North/south doors centered on x = RoomW/2
	for _, d := range []DoorDir{DoorNorth, DoorSouth} {
		tiles := doorTiles(d)
		mid := tiles[1][0]
		if mid != RoomW/2 {
			t.Errorf("door %d center x=%d, want %d", d, mid, RoomW/2)
		}
	}
	// West/east doors centered on z = RoomH/2
	for _, d := range []DoorDir{DoorWest, DoorEast} {
		tiles := doorTiles(d)
		mid := tiles[1][1]
		if mid != RoomH/2 {
			t.Errorf("door %d center z=%d, want %d", d, mid, RoomH/2)
		}
	}
}

func TestDoorTilesInvalidDir(t *testing.T) {
	tiles := doorTiles(DoorDir(99))
	if tiles != nil {
		t.Errorf("invalid door dir should return nil, got %v", tiles)
	}
}

// ---------- NeighborCoord ----------

func TestNeighborCoordDirections(t *testing.T) {
	origin := RoomCoord{5, 5}
	cases := []struct {
		dir  DoorDir
		want RoomCoord
	}{
		{DoorNorth, RoomCoord{5, 4}},
		{DoorSouth, RoomCoord{5, 6}},
		{DoorEast, RoomCoord{6, 5}},
		{DoorWest, RoomCoord{4, 5}},
	}
	for _, c := range cases {
		got := NeighborCoord(origin, c.dir)
		if got != c.want {
			t.Errorf("NeighborCoord dir=%d: want %v, got %v", c.dir, c.want, got)
		}
	}
}

func TestNeighborCoordRoundTrip(t *testing.T) {
	origin := RoomCoord{3, 7}
	for d := DoorDir(0); d < DoorCount; d++ {
		neighbor := NeighborCoord(origin, d)
		back := NeighborCoord(neighbor, d.Opposite())
		if back != origin {
			t.Errorf("round trip dir=%d: started at %v, ended at %v", d, origin, back)
		}
	}
}

func TestNeighborCoordNegative(t *testing.T) {
	origin := RoomCoord{0, 0}
	got := NeighborCoord(origin, DoorNorth)
	if got.Z != -1 {
		t.Errorf("north from origin: want Z=-1, got Z=%d", got.Z)
	}
	got = NeighborCoord(origin, DoorWest)
	if got.X != -1 {
		t.Errorf("west from origin: want X=-1, got X=%d", got.X)
	}
}

// ---------- SpawnPos ----------

func TestSpawnPosInsideRoom(t *testing.T) {
	for d := DoorDir(0); d < DoorCount; d++ {
		x, z := SpawnPos(d)
		if x < 1 || x > float32(RoomW)-1 {
			t.Errorf("SpawnPos dir=%d: x=%f out of room", d, x)
		}
		if z < 1 || z > float32(RoomH)-1 {
			t.Errorf("SpawnPos dir=%d: z=%f out of room", d, z)
		}
	}
}

func TestSpawnPosNearCorrectWall(t *testing.T) {
	x, z := SpawnPos(DoorNorth)
	if z > 3 {
		t.Errorf("north spawn z=%f, should be near top", z)
	}
	x, z = SpawnPos(DoorSouth)
	if z < float32(RoomH)-3 {
		t.Errorf("south spawn z=%f, should be near bottom", z)
	}
	x, z = SpawnPos(DoorWest)
	if x > 3 {
		t.Errorf("west spawn x=%f, should be near left", x)
	}
	x, z = SpawnPos(DoorEast)
	_ = z
	if x < float32(RoomW)-3 {
		t.Errorf("east spawn x=%f, should be near right", x)
	}
}

// ---------- GenerateRoom ----------

func TestGenerateRoomStartRoomAllDoorsOpen(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{0, 0}, 0, noDoors)

	for d := DoorDir(0); d < DoorCount; d++ {
		if !room.Doors[d] {
			t.Errorf("start room missing door %d", d)
		}
	}
	// All door tiles should be TileDoorOpen
	for d := DoorDir(0); d < DoorCount; d++ {
		for _, tile := range doorTiles(d) {
			if room.Tiles[tile[1]][tile[0]] != TileDoorOpen {
				t.Errorf("start room door %d tile (%d,%d) not open", d, tile[0], tile[1])
			}
		}
	}
}

func TestGenerateRoomStartRoomCleared(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{0, 0}, 0, noDoors)
	if !room.Cleared {
		t.Error("start room should be cleared")
	}
}

func TestGenerateRoomStartRoomNoEnemies(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{0, 0}, 0, noDoors)
	if len(room.Spawns) != 0 {
		t.Errorf("start room should have no enemies, got %d", len(room.Spawns))
	}
}

func TestGenerateRoomNonStartDoorsAreClosed(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var required [DoorCount]bool
	required[DoorSouth] = true
	room := GenerateRoom(rng, RoomCoord{1, 0}, 3, required)

	for d := DoorDir(0); d < DoorCount; d++ {
		if !room.Doors[d] {
			continue
		}
		for _, tile := range doorTiles(d) {
			if room.Tiles[tile[1]][tile[0]] != TileDoorClosed {
				t.Errorf("non-start room door %d tile (%d,%d) should be closed, got %d",
					d, tile[0], tile[1], room.Tiles[tile[1]][tile[0]])
			}
		}
	}
}

func TestGenerateRoomRequiredDoorsAlwaysPresent(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 100; i++ {
		var required [DoorCount]bool
		required[DoorWest] = true
		required[DoorEast] = true
		room := GenerateRoom(rng, RoomCoord{5, 5}, 5, required)
		if !room.Doors[DoorWest] {
			t.Fatalf("required DoorWest missing on iteration %d", i)
		}
		if !room.Doors[DoorEast] {
			t.Fatalf("required DoorEast missing on iteration %d", i)
		}
	}
}

func TestGenerateRoomHasAtLeastOneDoor(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 200; i++ {
		var noDoors [DoorCount]bool
		room := GenerateRoom(rng, RoomCoord{3, 3}, i%15, noDoors)
		hasDoor := false
		for d := DoorDir(0); d < DoorCount; d++ {
			if room.Doors[d] {
				hasDoor = true
				break
			}
		}
		if !hasDoor {
			t.Fatalf("room at iteration %d has no doors", i)
		}
	}
}

func TestGenerateRoomWallBorder(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{2, 2}, 5, noDoors)

	// Non-door border tiles should be walls
	doorSet := map[[2]int]bool{}
	for d := DoorDir(0); d < DoorCount; d++ {
		if room.Doors[d] {
			for _, tile := range doorTiles(d) {
				doorSet[tile] = true
			}
		}
	}

	for x := 0; x < RoomW; x++ {
		for _, z := range []int{0, RoomH - 1} {
			if doorSet[[2]int{x, z}] {
				continue
			}
			if room.Tiles[z][x] != TileWall {
				t.Errorf("border tile (%d,%d) should be wall, got %d", x, z, room.Tiles[z][x])
			}
		}
	}
	for z := 0; z < RoomH; z++ {
		for _, x := range []int{0, RoomW - 1} {
			if doorSet[[2]int{x, z}] {
				continue
			}
			if room.Tiles[z][x] != TileWall {
				t.Errorf("border tile (%d,%d) should be wall, got %d", x, z, room.Tiles[z][x])
			}
		}
	}
}

func TestGenerateRoomInteriorIsFloorOrPillar(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{2, 2}, 10, noDoors)

	for z := 1; z < RoomH-1; z++ {
		for x := 1; x < RoomW-1; x++ {
			tile := room.Tiles[z][x]
			if tile != TileFloor && tile != TilePillar {
				t.Errorf("interior tile (%d,%d) unexpected type %d", x, z, tile)
			}
		}
	}
}

func TestGenerateRoomEnemyCount(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for depth := 0; depth < 20; depth++ {
		var required [DoorCount]bool
		required[DoorNorth] = true
		room := GenerateRoom(rng, RoomCoord{1, 1}, depth, required)

		baseCount := 2 + depth/3
		if baseCount > 5 {
			baseCount = 5
		}
		minExpected := baseCount
		maxExpected := baseCount + 1 // +rng.Intn(2) adds 0 or 1

		if len(room.Spawns) < minExpected || len(room.Spawns) > maxExpected {
			t.Errorf("depth %d: expected %d-%d enemies, got %d",
				depth, minExpected, maxExpected, len(room.Spawns))
		}
	}
}

func TestGenerateRoomEnemyTypesAtDepthZero(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var required [DoorCount]bool
	required[DoorNorth] = true

	for i := 0; i < 50; i++ {
		room := GenerateRoom(rng, RoomCoord{1, 1}, 0, required)
		for _, s := range room.Spawns {
			if s.Type != EnemyMinion {
				t.Fatalf("depth 0 should only spawn minions, got type %d", s.Type)
			}
		}
	}
}

func TestGenerateRoomEnemyTypesDeepFloors(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var required [DoorCount]bool
	required[DoorNorth] = true

	typeSeen := map[EnemyType]bool{}
	for i := 0; i < 200; i++ {
		room := GenerateRoom(rng, RoomCoord{1, 1}, 10, required)
		for _, s := range room.Spawns {
			typeSeen[s.Type] = true
		}
	}
	if !typeSeen[EnemyMinion] || !typeSeen[EnemyWarrior] || !typeSeen[EnemyMage] {
		t.Errorf("at depth 10, expected all enemy types to appear: %v", typeSeen)
	}
}

func TestGenerateRoomEnemiesOnFloorTiles(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		var required [DoorCount]bool
		required[DoorSouth] = true
		room := GenerateRoom(rng, RoomCoord{1, 1}, 5, required)
		for _, s := range room.Spawns {
			// Spawns are at x+0.5, z+0.5 so the tile is at floor(x), floor(z)
			tx, tz := int(s.X), int(s.Z)
			if tx < 3 || tx >= RoomW-3 || tz < 3 || tz >= RoomH-3 {
				t.Errorf("enemy spawn (%d,%d) outside safe zone", tx, tz)
			}
		}
	}
}

func TestGenerateRoomDepthStored(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{0, 0}, 7, noDoors)
	if room.Depth != 7 {
		t.Errorf("room depth: want 7, got %d", room.Depth)
	}
}

func TestGenerateRoomPropsNotNil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var noDoors [DoorCount]bool
	room := GenerateRoom(rng, RoomCoord{0, 0}, 0, noDoors)
	if room.Props == nil {
		t.Error("start room props map should not be nil")
	}
}

func TestGenerateRoomPropsNotOnDoorTiles(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		var required [DoorCount]bool
		required[DoorNorth] = true
		room := GenerateRoom(rng, RoomCoord{1, 1}, 5, required)

		doorSet := map[[2]int]bool{}
		for d := DoorDir(0); d < DoorCount; d++ {
			if room.Doors[d] {
				for _, tile := range doorTiles(d) {
					doorSet[tile] = true
				}
			}
		}

		for pos := range room.Props {
			if doorSet[pos] {
				t.Errorf("prop placed on door tile %v", pos)
			}
		}
	}
}

func TestGenerateRoomPillarsDontBlockDoors(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 200; i++ {
		var required [DoorCount]bool
		required[DoorNorth] = true
		required[DoorSouth] = true
		room := GenerateRoom(rng, RoomCoord{1, 1}, 10, required)

		for d := DoorDir(0); d < DoorCount; d++ {
			if !room.Doors[d] {
				continue
			}
			for _, tile := range doorTiles(d) {
				if room.Tiles[tile[1]][tile[0]] == TilePillar {
					t.Errorf("pillar on door tile %v for door %d", tile, d)
				}
			}
		}
	}
}

func TestGenerateRoomNonStartNotCleared(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	var required [DoorCount]bool
	required[DoorNorth] = true
	room := GenerateRoom(rng, RoomCoord{1, 1}, 3, required)
	if room.Cleared {
		t.Error("non-start room should not be cleared")
	}
}

// ---------- wallRotation ----------

func TestWallRotationValues(t *testing.T) {
	cases := []struct {
		dir  int
		want float32
	}{
		{0, 180}, {1, 0}, {2, 90}, {3, -90},
	}
	for _, c := range cases {
		got := wallRotation(c.dir)
		if got != c.want {
			t.Errorf("wallRotation(%d): want %f, got %f", c.dir, c.want, got)
		}
	}
}

func TestWallRotationDefault(t *testing.T) {
	got := wallRotation(99)
	if got != 0 {
		t.Errorf("wallRotation(99): want 0, got %f", got)
	}
}

// ---------- Determinism ----------

func TestGenerateRoomDeterministic(t *testing.T) {
	gen := func(seed int64) *RoomDef {
		rng := rand.New(rand.NewSource(seed))
		var required [DoorCount]bool
		required[DoorWest] = true
		return GenerateRoom(rng, RoomCoord{3, 3}, 5, required)
	}

	r1 := gen(12345)
	r2 := gen(12345)

	if r1.Tiles != r2.Tiles {
		t.Error("same seed should produce same tiles")
	}
	if len(r1.Spawns) != len(r2.Spawns) {
		t.Error("same seed should produce same spawn count")
	}
	for i := range r1.Spawns {
		if r1.Spawns[i] != r2.Spawns[i] {
			t.Errorf("spawn %d differs between runs", i)
		}
	}
}
