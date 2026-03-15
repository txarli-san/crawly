package main

import (
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// Input abstraction so game.go doesn't import raylib directly
const (
	keyW = 0
	keyA = 1
	keyS = 2
	keyD = 3
)

func isKeyDown(key int) bool {
	switch key {
	case keyW:
		return rl.IsKeyDown(rl.KeyW)
	case keyA:
		return rl.IsKeyDown(rl.KeyA)
	case keyS:
		return rl.IsKeyDown(rl.KeyS)
	case keyD:
		return rl.IsKeyDown(rl.KeyD)
	}
	return false
}

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Crawly — Twin-Stick Dungeon Crawler")
	defer rl.CloseWindow()
	rl.SetExitKey(0)
	rl.SetTargetFPS(60)

	// --- Shader ---
	shader := rl.LoadShaderFromMemory(lightingVS, lightingFS)
	defer rl.UnloadShader(shader)

	locLightDir := rl.GetShaderLocation(shader, "lightDir")
	locLightColor := rl.GetShaderLocation(shader, "lightColor")
	locAmbientColor := rl.GetShaderLocation(shader, "ambientColor")
	locViewPos := rl.GetShaderLocation(shader, "viewPos")

	// Top-down lighting — light from above and slightly forward
	rl.SetShaderValue(shader, locLightDir, []float32{-0.2, -0.9, -0.3}, rl.ShaderUniformVec3)
	rl.SetShaderValue(shader, locLightColor, []float32{1.0, 0.95, 0.85, 1.0}, rl.ShaderUniformVec4)
	rl.SetShaderValue(shader, locAmbientColor, []float32{0.5, 0.5, 0.55, 1.0}, rl.ShaderUniformVec4)

	shader.UpdateLocation(rl.ShaderLocMatrixModel, rl.GetShaderLocation(shader, "matModel"))
	shader.UpdateLocation(rl.ShaderLocMatrixNormal, rl.GetShaderLocation(shader, "matNormal"))

	// --- Models ---
	floorModel := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_large.gltf.glb")
	defer rl.UnloadModel(floorModel)
	applyShaderToModel(floorModel, shader)

	floorCrackedA := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_largeCrackedA.gltf.glb")
	defer rl.UnloadModel(floorCrackedA)
	applyShaderToModel(floorCrackedA, shader)

	floorCrackedB := rl.LoadModel("assets/models/dungeon/floors/tileBrickB_largeCrackedB.gltf.glb")
	defer rl.UnloadModel(floorCrackedB)
	applyShaderToModel(floorCrackedB, shader)

	wallModel := rl.LoadModel("assets/models/dungeon/walls/wall.gltf.glb")
	defer rl.UnloadModel(wallModel)
	applyShaderToModel(wallModel, shader)

	// Animated character models
	mageAnim := loadAnimatedModel("assets/models/characters/animated/Mage.glb")
	defer mageAnim.Unload()
	mageAnim.GearBindings = gearBindings("Mage")

	knightAnim := loadAnimatedModel("assets/models/characters/animated/Knight.glb")
	defer knightAnim.Unload()
	knightAnim.GearBindings = gearBindings("Fighter")

	heroModels := map[PlayerClass]*AnimatedModel{
		ClassMage:    mageAnim,
		ClassWarrior: knightAnim,
	}

	skelMinionAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Minion.glb")
	defer skelMinionAnim.Unload()
	skelWarriorAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Warrior.glb")
	defer skelWarriorAnim.Unload()
	skelMageAnim := loadAnimatedModel("assets/models/characters/animated/Skeleton_Mage.glb")
	defer skelMageAnim.Unload()

	skelModels := map[EnemyType]*AnimatedModel{
		EnemyMinion:  skelMinionAnim,
		EnemyWarrior: skelWarriorAnim,
		EnemyMage:    skelMageAnim,
	}

	// --- Measurements ---
	floorBBox := rl.GetModelBoundingBox(floorModel)
	tileUnit := floorBBox.Max.X - floorBBox.Min.X
	floorSurfaceY := floorBBox.Max.Y

	wallBBox := rl.GetModelBoundingBox(wallModel)
	wallWidth := wallBBox.Max.X - wallBBox.Min.X
	wallScale := tileUnit / wallWidth

	propModels := LoadPropModels(shader, wallScale)
	defer propModels.Unload()

	charBBox := rl.GetModelBoundingBox(*mageAnim.Model)
	charScale := (tileUnit * 1.4) / (charBBox.Max.X - charBBox.Min.X)
	charYOffset := floorSurfaceY - charBBox.Min.Y*charScale

	// --- Camera: isometric-style angle (~55° elevation) ---
	roomCenterX := float32(RoomW) * tileUnit / 2.0
	roomCenterZ := float32(RoomH) * tileUnit / 2.0

	// Camera distance calculated to fit the room width on screen
	camElevation := float32(55.0) * math.Pi / 180.0 // 55° from ground
	camDist := float32(RoomW) * tileUnit * 1.0 // distance from target
	camHeight := camDist * float32(math.Sin(float64(camElevation)))
	camOffsetZ := camDist * float32(math.Cos(float64(camElevation)))

	camera := rl.Camera3D{
		Position:   rl.Vector3{X: roomCenterX, Y: camHeight, Z: roomCenterZ + camOffsetZ},
		Target:     rl.Vector3{X: roomCenterX, Y: 0, Z: roomCenterZ},
		Up:         rl.Vector3{Y: 1},
		Fovy:       45,
		Projection: rl.CameraPerspective,
	}

	// --- Ray-ground intersection for mouse aiming ---
	rayHitGround := func(ray rl.Ray) (float32, float32, bool) {
		groundY := floorSurfaceY
		if ray.Direction.Y == 0 {
			return 0, 0, false
		}
		t := (groundY - ray.Position.Y) / ray.Direction.Y
		if t <= 0 {
			return 0, 0, false
		}
		return ray.Position.X + ray.Direction.X*t, ray.Position.Z + ray.Direction.Z*t, true
	}

	// --- Game state ---
	var game *GameState
	selectedClass := ClassMage

	initGame := func(class PlayerClass) {
		seed := time.Now().UnixNano()
		game = NewGame(seed, tileUnit, floorSurfaceY, class)
	}

	phase := PhaseTitle

	// --- Main Loop ---
	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// Update view pos for shader
		viewPos := []float32{camera.Position.X, camera.Position.Y, camera.Position.Z}
		rl.SetShaderValue(shader, locViewPos, viewPos, rl.ShaderUniformVec3)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{R: 8, G: 8, B: 12, A: 255})

		switch phase {
		case PhaseTitle:
			drawTitle(selectedClass)
			if rl.IsKeyPressed(rl.KeyA) || rl.IsKeyPressed(rl.KeyLeft) {
				selectedClass = (selectedClass + 1) % 2
			}
			if rl.IsKeyPressed(rl.KeyD) || rl.IsKeyPressed(rl.KeyRight) {
				selectedClass = (selectedClass + 1) % 2
			}
			if rl.IsKeyPressed(rl.KeySpace) {
				initGame(selectedClass)
				phase = PhasePlaying
			}

		case PhasePlaying, PhaseTransition:
			if game == nil {
				break
			}

			// Aiming: mouse → world position
			mouseScreen := rl.GetMousePosition()
			ray := rl.GetScreenToWorldRay(mouseScreen, camera)
			if wx, wz, ok := rayHitGround(ray); ok {
				p := &game.Player
				aimDX := wx - p.X
				aimDZ := wz - p.Z
				aimLen := float32(math.Sqrt(float64(aimDX*aimDX + aimDZ*aimDZ)))
				if aimLen > 0.01 {
					aimDX /= aimLen
					aimDZ /= aimLen
					p.FacingAngle = float32(math.Atan2(float64(aimDX), float64(aimDZ))) * 180 / math.Pi
				}

				// Attack
				if game.Phase == PhasePlaying && rl.IsMouseButtonDown(rl.MouseButtonLeft) {
					switch p.Class {
					case ClassMage:
						if p.FireTimer <= 0 {
							game.SpawnPlayerProjectiles(aimDX, aimDZ)
							p.FireTimer = 1.0 / p.Stats.FireRate
						}
					case ClassWarrior:
						if p.MeleeCooldown <= 0 && p.MeleeTimer <= 0 {
							game.StartMeleeAttack()
						}
					}
				}
			}

			// Dodge / Block
			if game.Phase == PhasePlaying {
				p := &game.Player
				switch p.Class {
				case ClassMage:
					// Mage: dodge roll
					if (rl.IsKeyPressed(rl.KeySpace) || rl.IsMouseButtonPressed(rl.MouseButtonRight)) && p.DodgeCooldown <= 0 && p.DodgeTimer <= 0 {
						var ddx, ddz float32
						if isKeyDown(keyW) {
							ddz -= 1
						}
						if isKeyDown(keyS) {
							ddz += 1
						}
						if isKeyDown(keyA) {
							ddx -= 1
						}
						if isKeyDown(keyD) {
							ddx += 1
						}
						if ddx == 0 && ddz == 0 {
							angle := p.FacingAngle * math.Pi / 180
							ddx = float32(math.Sin(float64(angle)))
							ddz = float32(math.Cos(float64(angle)))
						}
						length := float32(math.Sqrt(float64(ddx*ddx + ddz*ddz)))
						if length > 0 {
							ddx /= length
							ddz /= length
						}
						p.DodgeVX = ddx * dodgeSpeed * tileUnit
						p.DodgeVZ = ddz * dodgeSpeed * tileUnit
						p.DodgeTimer = dodgeDuration
						p.InvulnTimer = dodgeDuration + 0.1
						p.DodgeCooldown = dodgeCooldownTime
					}
				case ClassWarrior:
					// Warrior: hold RMB/Space to block, tap to parry
					if rl.IsKeyPressed(rl.KeySpace) || rl.IsMouseButtonPressed(rl.MouseButtonRight) {
						p.BlockTimer = 0.5
						p.ParryWindow = 0.15 // perfect parry in first 150ms
					}
				}
			}

			// Update game
			game.Update(dt)

			// Sync outer phase
			phase = game.Phase

			// Render 3D scene
			drawScene(
				camera, game, dt,
				tileUnit, floorSurfaceY, charYOffset, charScale, wallScale,
				floorModel, floorCrackedA, floorCrackedB, wallModel,
				heroModels, skelModels, propModels,
			)

			// Transition overlay
			if game.Phase == PhaseTransition {
				progress := 1.0 - game.TransitionTimer/0.5
				drawTransition(progress)
			}

		case PhaseDead:
			if game != nil {
				// Render frozen scene
				drawScene(
					camera, game, 0,
					tileUnit, floorSurfaceY, charYOffset, charScale, wallScale,
					floorModel, floorCrackedA, floorCrackedB, wallModel,
					heroModels, skelModels, propModels,
				)
				drawDeath(game)

				if game.DeathTimer <= 0 && rl.IsKeyPressed(rl.KeySpace) {
					initGame(selectedClass)
					phase = PhasePlaying
				}
			}
		}

		rl.EndDrawing()
	}
}
