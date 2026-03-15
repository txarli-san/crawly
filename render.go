package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"image/color"
	"math"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Lighting shader — identical to slumbgate
const lightingVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;

out vec3 fragPosition;
out vec2 fragTexCoord;
out vec3 fragNormal;
out vec4 fragColor;

void main() {
    fragPosition = vec3(matModel * vec4(vertexPosition, 1.0));
    fragTexCoord = vertexTexCoord;
    fragNormal = normalize(vec3(matNormal * vec4(vertexNormal, 0.0)));
    fragColor = vertexColor;
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const lightingFS = `#version 330
in vec3 fragPosition;
in vec2 fragTexCoord;
in vec3 fragNormal;
in vec4 fragColor;

uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir;
uniform vec4 lightColor;
uniform vec4 ambientColor;
uniform vec3 viewPos;

out vec4 finalColor;

void main() {
    vec4 texColor = texture(texture0, fragTexCoord);
    vec4 baseColor = texColor * colDiffuse * fragColor;

    float diff = max(dot(fragNormal, -lightDir), 0.0);
    vec4 diffuse = diff * lightColor;

    vec3 viewDir = normalize(viewPos - fragPosition);
    vec3 reflectDir = reflect(lightDir, fragNormal);
    float spec = pow(max(dot(viewDir, reflectDir), 0.0), 16.0);
    vec4 specular = spec * 0.3 * lightColor;

    finalColor = (ambientColor + diffuse + specular) * baseColor;
    finalColor.a = baseColor.a;
}
`

// PropModels holds all loaded prop models and per-model scales.
type PropModels struct {
	Models []rl.Model
	Scales []rl.Vector3
}

var propModelPaths = [PropCount]string{
	PropBarrel:             "props/barrel.gltf.glb",
	PropBarrelDark:         "props/barrelDark.gltf.glb",
	PropCrate:              "props/crate.gltf.glb",
	PropCrateDark:          "props/crateDark.gltf.glb",
	PropBucket:             "props/bucket.gltf.glb",
	PropPots:               "props/pots.gltf.glb",
	PropWeaponRack:         "props/weaponRack.gltf.glb",
	PropBench:              "props/bench.gltf.glb",
	PropTableMedium:        "props/tableMedium.gltf.glb",
	PropStool:              "props/stool.gltf.glb",
	PropBanner:             "props/banner.gltf.glb",
	PropBookcaseFilled:     "props/bookcaseFilled.gltf.glb",
	PropBookcaseWideFilled: "props/bookcaseWideFilled.gltf.glb",
	PropTableSmall:         "props/tableSmall.gltf.glb",
	PropBookA:              "props/bookA.gltf.glb",
	PropBookOpenA:          "props/bookOpenA.gltf.glb",
	PropSpellBook:          "props/spellBook.gltf.glb",
	PropTableLarge:         "props/tableLarge.gltf.glb",
	PropChair:              "props/chair.gltf.glb",
	PropMug:                "props/mug.gltf.glb",
	PropPlate:              "props/plate.gltf.glb",
	PropPlateFull:          "props/plateFull.gltf.glb",
	PropPillar:             "walls/pillar.gltf.glb",
	PropPillarBroken:       "walls/pillar_broken.gltf.glb",
	PropBricks:             "props/bricks.gltf.glb",
	PropFloorDecoShattered: "props/floorDecoration_shatteredBricks.gltf.glb",
	PropTileSpikes:         "hazards/tileSpikes.gltf.glb",
	PropTileSpikesLarge:    "hazards/tileSpikes_large.gltf.glb",
	PropTorchWall:          "hazards/torchWall.gltf.glb",
	PropFloorDecoTiles:     "props/floorDecoration_tilesSmall.gltf.glb",
}

var propScaleFactors = [PropCount]float32{
	PropBarrel: 1.0, PropBarrelDark: 1.0,
	PropCrate: 1.0, PropCrateDark: 1.0,
	PropBucket: 1.0, PropPots: 1.0,
	PropWeaponRack: 1.2, PropBench: 1.0,
	PropTableMedium: 1.0, PropStool: 1.0,
	PropBanner: 1.2, PropBookcaseFilled: 1.5,
	PropBookcaseWideFilled: 1.5, PropTableSmall: 1.0,
	PropBookA: 0.8, PropBookOpenA: 0.8, PropSpellBook: 0.8,
	PropTableLarge: 1.2, PropChair: 1.0,
	PropMug: 0.8, PropPlate: 0.8, PropPlateFull: 0.8,
	PropPillar: 1.0, PropPillarBroken: 1.0,
	PropBricks: 1.0, PropFloorDecoShattered: 1.0,
	PropTileSpikes: 1.0, PropTileSpikesLarge: 1.0,
	PropTorchWall: 1.0, PropFloorDecoTiles: 1.0,
}

func LoadPropModels(shader rl.Shader, wallScale float32) *PropModels {
	pm := &PropModels{
		Models: make([]rl.Model, PropCount),
		Scales: make([]rl.Vector3, PropCount),
	}
	for i := 0; i < PropCount; i++ {
		path := "assets/models/dungeon/" + propModelPaths[i]
		pm.Models[i] = rl.LoadModel(path)
		applyShaderToModel(pm.Models[i], shader)
		s := wallScale * propScaleFactors[i] * 3.0
		pm.Scales[i] = rl.Vector3{X: s, Y: s, Z: s}
	}
	return pm
}

func (pm *PropModels) Unload() {
	for i := range pm.Models {
		rl.UnloadModel(pm.Models[i])
	}
}

func applyShaderToModel(model rl.Model, shader rl.Shader) {
	materials := unsafe.Slice(model.Materials, model.MaterialCount)
	for i := range materials {
		materials[i].Shader = shader
	}
}

// AnimatedModel holds a model with its animation set and name→index lookup.
type AnimatedModel struct {
	Model        *rl.Model
	Anims        []rl.ModelAnimation
	Index        map[string]int
	GearBindings []GearBinding
}

func loadAnimatedModel(path string) *AnimatedModel {
	model := rl.LoadModel(path)
	meshes := unsafe.Slice(model.Meshes, model.MeshCount)
	for i := range meshes {
		n := meshes[i].VertexCount
		if meshes[i].BoneWeights == nil {
			meshes[i].BoneWeights = (*float32)(C.calloc(C.size_t(n*4), C.size_t(unsafe.Sizeof(float32(0)))))
		}
		if meshes[i].BoneIds == nil {
			meshes[i].BoneIds = (*int32)(C.calloc(C.size_t(n*4), C.size_t(unsafe.Sizeof(int32(0)))))
		}
	}
	m := &AnimatedModel{
		Model: &model,
		Anims: rl.LoadModelAnimations(path),
		Index: map[string]int{},
	}
	for i, a := range m.Anims {
		name := ""
		for _, b := range a.Name {
			if b == 0 {
				break
			}
			name += string(rune(b))
		}
		m.Index[name] = i
	}
	return m
}

type GearBinding struct {
	Bone int
}

func gearBindings(class string) []GearBinding {
	none := GearBinding{-1}
	switch class {
	case "Fighter":
		return []GearBinding{
			{8}, {8}, {8}, {8}, {8}, {13}, {13}, {14}, {3},
			none, none, none, none, none, none,
		}
	case "Mage":
		return []GearBinding{
			{8}, {8}, {13}, {13}, {14}, {3},
			none, none, none, none, none, none,
		}
	}
	return nil
}

type AnimState struct {
	Clip  string
	Frame int32
	Time  float32
	Loop  bool
	Done  bool
}

func (m *AnimatedModel) DrawFiltered(visible []bool, pos, rotAxis rl.Vector3, rotAngle float32, scale rl.Vector3, tint color.RGBA) {
	if visible == nil {
		rl.DrawModelEx(*m.Model, pos, rotAxis, rotAngle, scale, tint)
		return
	}

	model := *m.Model
	transform := rl.MatrixMultiply(
		rl.MatrixMultiply(
			rl.MatrixScale(scale.X, scale.Y, scale.Z),
			rl.MatrixRotate(rotAxis, rotAngle*math.Pi/180),
		),
		rl.MatrixTranslate(pos.X, pos.Y, pos.Z),
	)
	transform = rl.MatrixMultiply(model.Transform, transform)

	meshes := unsafe.Slice(model.Meshes, model.MeshCount)
	materials := unsafe.Slice(model.Materials, model.MaterialCount)
	meshMats := unsafe.Slice(model.MeshMaterial, model.MeshCount)

	for i := int32(0); i < model.MeshCount; i++ {
		if i < int32(len(visible)) && !visible[i] {
			continue
		}
		meshXform := transform
		if int(i) < len(m.GearBindings) && m.GearBindings[i].Bone >= 0 {
			gb := m.GearBindings[i]
			boneMats := unsafe.Slice(meshes[i].BoneMatrices, meshes[i].BoneCount)
			boneMat := boneMats[gb.Bone]
			meshXform = rl.MatrixMultiply(boneMat, transform)
		}
		matIdx := meshMats[i]
		mat := materials[matIdx]
		diffuse := mat.GetMap(rl.MapAlbedo)
		origColor := diffuse.Color
		diffuse.Color = tint
		rl.DrawMesh(meshes[i], mat, meshXform)
		diffuse.Color = origColor
	}
}

func (m *AnimatedModel) Unload() {
	rl.UnloadModel(*m.Model)
	rl.UnloadModelAnimations(m.Anims)
}

func (m *AnimatedModel) UpdateAnim(anim *AnimState, dt float32) {
	idx, ok := m.Index[anim.Clip]
	if !ok || len(m.Anims) == 0 {
		return
	}
	a := m.Anims[idx]
	duration := float32(a.FrameCount) / 60.0
	anim.Time += dt
	if anim.Loop {
		for anim.Time >= duration {
			anim.Time -= duration
		}
	} else if anim.Time >= duration {
		anim.Time = duration
		anim.Done = true
	}
	anim.Frame = int32(anim.Time / duration * float32(a.FrameCount))
	if anim.Frame >= a.FrameCount {
		anim.Frame = a.FrameCount - 1
	}
	rl.UpdateModelAnimation(*m.Model, a, anim.Frame)
}

// ---------- Drawing ----------

func drawScene(
	camera rl.Camera3D,
	game *GameState,
	dt float32,
	tileUnit, floorSurfaceY, charYOffset, charScale, wallScale float32,
	floorModel rl.Model,
	floorCrackedA, floorCrackedB rl.Model,
	wallModel rl.Model,
	heroModels map[PlayerClass]*AnimatedModel,
	skelModels map[EnemyType]*AnimatedModel,
	propModels *PropModels,
) {
	rl.BeginMode3D(camera)

	ones := rl.Vector3{X: 1, Y: 1, Z: 1}
	wallScaleVec := rl.Vector3{X: wallScale, Y: wallScale, Z: wallScale}
	halfTile := tileUnit * 0.5
	charScaleVec := rl.Vector3{X: charScale, Y: charScale, Z: charScale}

	tileToWorld := func(tx, tz int) rl.Vector3 {
		return rl.Vector3{
			X: (float32(tx) + 0.5) * tileUnit,
			Z: (float32(tz) + 0.5) * tileUnit,
		}
	}

	floorVariant := func(x, z int) rl.Model {
		h := (x*7 + z*13 + x*z*3) % 10
		if h < 0 {
			h += 10
		}
		if h == 0 {
			return floorCrackedA
		}
		if h == 1 {
			return floorCrackedB
		}
		return floorModel
	}

	room := game.CurrentRoom
	if room == nil {
		rl.EndMode3D()
		return
	}

	// Draw room tiles
	for tz := 0; tz < RoomH; tz++ {
		for tx := 0; tx < RoomW; tx++ {
			tile := room.Tiles[tz][tx]
			pos := tileToWorld(tx, tz)

			switch tile {
			case TileFloor, TileDoorOpen:
				fm := floorVariant(tx, tz)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.White)

			case TileDoorClosed:
				fm := floorVariant(tx, tz)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.Color{R: 80, G: 60, B: 60, A: 255})
				// Draw a barrier
				barrierPos := pos
				barrierPos.Y = floorSurfaceY + tileUnit*0.3
				rl.DrawCubeV(barrierPos, rl.Vector3{X: tileUnit * 0.9, Y: tileUnit * 0.6, Z: tileUnit * 0.9},
					rl.Color{R: 120, G: 80, B: 40, A: 200})

			case TileWall:
				// Render walls on edges facing floor tiles
				drawWall := func(wallPos rl.Vector3, rotation float32) {
					wallPos.Y = floorSurfaceY
					rl.DrawModelEx(wallModel, wallPos, rl.Vector3{Y: 1}, rotation, wallScaleVec, rl.White)
				}
				isOpen := func(nx, nz int) bool {
					if nx < 0 || nx >= RoomW || nz < 0 || nz >= RoomH {
						return false
					}
					t := room.Tiles[nz][nx]
					return t == TileFloor || t == TileDoorOpen || t == TileDoorClosed || t == TilePillar
				}
				if isOpen(tx, tz-1) {
					drawWall(rl.Vector3{X: pos.X, Z: pos.Z - halfTile}, 180)
				}
				if isOpen(tx, tz+1) {
					drawWall(rl.Vector3{X: pos.X, Z: pos.Z + halfTile}, 0)
				}
				if isOpen(tx-1, tz) {
					drawWall(rl.Vector3{X: pos.X - halfTile, Z: pos.Z}, -90)
				}
				if isOpen(tx+1, tz) {
					drawWall(rl.Vector3{X: pos.X + halfTile, Z: pos.Z}, 90)
				}

			case TilePillar:
				// Floor underneath
				fm := floorVariant(tx, tz)
				rl.DrawModelEx(fm, pos, rl.Vector3{Y: 1}, 0, ones, rl.Color{R: 180, G: 180, B: 180, A: 255})
				// Pillar cube
				pillarPos := pos
				pillarPos.Y = floorSurfaceY + tileUnit*0.5
				rl.DrawCubeV(pillarPos, rl.Vector3{X: tileUnit * 0.5, Y: tileUnit * 1.0, Z: tileUnit * 0.5},
					rl.Color{R: 100, G: 95, B: 90, A: 255})
			}
		}
	}

	// Draw props
	if room.Props != nil && propModels != nil {
		halfTileOff := tileUnit * 0.35
		for key, prop := range room.Props {
			propPos := tileToWorld(key[0], key[1])
			propPos.Y = floorSurfaceY
			switch prop.Wall {
			case 0:
				propPos.Z -= halfTileOff
			case 1:
				propPos.Z += halfTileOff
			case 2:
				propPos.X -= halfTileOff
			case 3:
				propPos.X += halfTileOff
			}
			rl.DrawModelEx(propModels.Models[prop.Model], propPos, rl.Vector3{Y: 1}, prop.Rotation, propModels.Scales[prop.Model], rl.White)
		}
	}

	// Draw enemies
	for i := range game.Enemies {
		e := &game.Enemies[i]
		if !e.Alive {
			continue
		}
		ePos := rl.Vector3{X: e.X, Y: charYOffset, Z: e.Z}

		// Tint from status effects and awareness state
		tint := rl.White
		if e.FireTimer > 0 {
			tint = rl.Color{R: 255, G: 150, B: 80, A: 255}
		} else if e.IceTimer > 0 {
			tint = rl.Color{R: 150, G: 200, B: 255, A: 255}
		} else if e.PoisonTimer > 0 {
			tint = rl.Color{R: 100, G: 255, B: 100, A: 255}
		} else if e.State == StateIdle {
			tint = rl.Color{R: 180, G: 180, B: 200, A: 255} // unaware = muted
		}

		wantClip := "Idle"
		if e.State == StateIdle {
			if e.Moving {
				wantClip = "Walking_A"
			} else {
				wantClip = "Idle"
			}
		} else {
			wantClip = "Idle_Combat"
			if e.Moving {
				wantClip = "Walking_A"
			}
		}
		if e.Anim.Clip != wantClip {
			e.Anim = AnimState{Clip: wantClip, Loop: true}
		}

		if am, ok := skelModels[e.Type]; ok {
			am.UpdateAnim(&e.Anim, dt)
			rl.DrawModelEx(*am.Model, ePos, rl.Vector3{Y: 1}, e.FacingAngle, charScaleVec, tint)
		}

		// HP bar
		if e.HP < e.MaxHP {
			barPos := rl.Vector3{X: e.X, Y: charYOffset + tileUnit*0.8, Z: e.Z}
			screenPos := rl.GetWorldToScreen(barPos, camera)
			barW := float32(30)
			barH := float32(4)
			ratio := float32(e.HP) / float32(e.MaxHP)
			rl.DrawRectangle(int32(screenPos.X-barW/2), int32(screenPos.Y), int32(barW), int32(barH),
				rl.Color{R: 40, G: 40, B: 40, A: 200})
			rl.DrawRectangle(int32(screenPos.X-barW/2), int32(screenPos.Y), int32(barW*ratio), int32(barH),
				rl.Color{R: 255, G: 60, B: 60, A: 255})
		}
	}

	// Draw player
	p := &game.Player
	playerPos := rl.Vector3{X: p.X, Y: charYOffset, Z: p.Z}

	// Invulnerability flash
	playerTint := rl.Color{R: 200, G: 220, B: 255, A: 255}
	if p.InvulnTimer > 0 {
		if int(p.InvulnTimer*10)%2 == 0 {
			playerTint = rl.Color{R: 255, G: 255, B: 255, A: 120}
		}
	}

	wantClip := "Idle"
	if p.Moving {
		wantClip = "Walking_A"
	}
	if p.Anim.Clip != wantClip {
		p.Anim = AnimState{Clip: wantClip, Loop: true}
	}
	heroModel := heroModels[game.Player.Class]
	heroModel.UpdateAnim(&p.Anim, dt)
	heroModel.DrawFiltered(p.VisibleMeshes, playerPos, rl.Vector3{Y: 1}, p.FacingAngle, charScaleVec, playerTint)

	// Melee arc visualization
	if p.MeleeTimer > 0 && p.Class != ClassMage {
		arcAlpha := uint8(float32(150) * (p.MeleeTimer / 0.2))
		arcRange := p.Stats.MeleeRange * tileUnit
		halfArc := p.Stats.MeleeArc / 2.0
		segments := 8
		baseAngle := p.FacingAngle * math.Pi / 180
		for s := 0; s < segments; s++ {
			a1 := baseAngle + float32(math.Pi/180)*(-halfArc+halfArc*2*float32(s)/float32(segments))
			a2 := baseAngle + float32(math.Pi/180)*(-halfArc+halfArc*2*float32(s+1)/float32(segments))
			x1 := p.X + float32(math.Sin(float64(a1)))*arcRange
			z1 := p.Z + float32(math.Cos(float64(a1)))*arcRange
			x2 := p.X + float32(math.Sin(float64(a2)))*arcRange
			z2 := p.Z + float32(math.Cos(float64(a2)))*arcRange
			arcY := floorSurfaceY + 0.03
			v1 := rl.Vector3{X: p.X, Y: arcY, Z: p.Z}
			v2 := rl.Vector3{X: x1, Y: arcY, Z: z1}
			v3 := rl.Vector3{X: x2, Y: arcY, Z: z2}
			arcCol := rl.Color{R: 255, G: 200, B: 100, A: arcAlpha}
			rl.DrawTriangle3D(v1, v3, v2, arcCol)
		}
	}

	// Draw projectiles
	for i := range game.Projectiles {
		proj := &game.Projectiles[i]
		if !proj.Alive {
			continue
		}
		projPos := rl.Vector3{X: proj.X, Y: floorSurfaceY + tileUnit*0.3, Z: proj.Z}
		size := proj.Radius * 2
		if proj.Owner == 0 {
			// Player projectile color
			col := rl.Color{R: 255, G: 255, B: 200, A: 255}
			if proj.Fire {
				col = rl.Color{R: 255, G: 120, B: 40, A: 255}
			} else if proj.Ice {
				col = rl.Color{R: 100, G: 200, B: 255, A: 255}
			} else if proj.Poison {
				col = rl.Color{R: 80, G: 255, B: 80, A: 255}
			}
			rl.DrawSphere(projPos, size, col)
			// Glow
			rl.DrawSphere(projPos, size*1.5, rl.Color{R: col.R, G: col.G, B: col.B, A: 60})
		} else {
			// Enemy projectile
			rl.DrawSphere(projPos, size, rl.Color{R: 255, G: 60, B: 60, A: 255})
		}
	}

	// Draw item drops
	for i := range game.ItemDrops {
		drop := &game.ItemDrops[i]
		if drop.Collected {
			continue
		}
		bob := float32(math.Sin(float64(drop.Timer)*3.0)) * tileUnit * 0.08
		dropPos := rl.Vector3{X: drop.X, Y: floorSurfaceY + tileUnit*0.25 + bob, Z: drop.Z}

		// Color by rarity
		col := rl.Color{R: 200, G: 200, B: 200, A: 255}
		if drop.Item != nil {
			switch drop.Item.Rarity {
			case 0: // health potion
				col = rl.Color{R: 60, G: 255, B: 80, A: 255}
			case 1:
				col = rl.Color{R: 180, G: 180, B: 180, A: 255}
			case 2:
				col = rl.Color{R: 80, G: 200, B: 255, A: 255}
			case 3:
				col = rl.Color{R: 255, G: 200, B: 50, A: 255}
			}
		}
		rl.DrawCube(dropPos, tileUnit*0.25, tileUnit*0.25, tileUnit*0.25, col)
		// Glow
		glowPos := dropPos
		glowPos.Y = floorSurfaceY + 0.02
		pulse := float32(math.Sin(float64(drop.Timer)*2.0))*0.3 + 0.7
		rl.DrawCubeV(glowPos, rl.Vector3{X: tileUnit * 0.5 * pulse, Y: 0.02, Z: tileUnit * 0.5 * pulse},
			rl.Color{R: col.R, G: col.G, B: col.B, A: uint8(40 * pulse)})
	}

	// Explosions
	for i := range game.Explosions {
		exp := &game.Explosions[i]
		if exp.Timer <= 0 {
			continue
		}
		progress := 1.0 - exp.Timer/exp.MaxTime
		radius := exp.Radius * (0.5 + float32(progress)*0.5)
		alpha := uint8((1.0 - progress) * 200)
		expPos := rl.Vector3{X: exp.X, Y: floorSurfaceY + tileUnit*0.2, Z: exp.Z}
		rl.DrawSphere(expPos, radius, rl.Color{R: 255, G: 160, B: 40, A: alpha})
		rl.DrawSphere(expPos, radius*0.6, rl.Color{R: 255, G: 255, B: 200, A: alpha})
	}

	rl.EndMode3D()

	// --- 2D HUD ---

	// HP bar
	barX, barY := int32(20), int32(screenHeight-60)
	barW, barH := int32(200), int32(20)
	hpRatio := float32(p.HP) / float32(p.MaxHP)
	rl.DrawRectangle(barX, barY, barW, barH, rl.Color{R: 40, G: 40, B: 40, A: 200})
	rl.DrawRectangle(barX, barY, int32(float32(barW)*hpRatio), barH, rl.Color{R: 220, G: 40, B: 40, A: 255})
	rl.DrawRectangleLines(barX, barY, barW, barH, rl.Color{R: 200, G: 200, B: 200, A: 255})
	rl.DrawText(fmt.Sprintf("HP: %d/%d", p.HP, p.MaxHP), barX+6, barY+3, 14, rl.White)

	// Shield indicator
	if p.Stats.ShieldStacks > 0 {
		rl.DrawText(fmt.Sprintf("Shield: %d", p.Stats.ShieldStacks), barX, barY-18, 14,
			rl.Color{R: 100, G: 200, B: 255, A: 255})
	}

	// Items collected — bottom right
	itemX := int32(screenWidth - 260)
	itemY := int32(screenHeight - 30)
	rl.DrawText(fmt.Sprintf("Items: %d", len(p.Items)), itemX, itemY, 14, rl.Color{R: 200, G: 200, B: 200, A: 255})

	// Item list — right side
	if len(p.Items) > 0 {
		panelX := int32(screenWidth - 220)
		panelY := int32(10)
		maxShow := 20
		if len(p.Items) < maxShow {
			maxShow = len(p.Items)
		}
		panelH := int32(8 + maxShow*16)
		rl.DrawRectangle(panelX-4, panelY-4, 214, panelH+8, rl.Color{R: 10, G: 10, B: 20, A: 180})
		for i := len(p.Items) - 1; i >= 0 && (len(p.Items)-1-i) < maxShow; i-- {
			item := p.Items[i]
			row := len(p.Items) - 1 - i
			col := rl.Color{R: 180, G: 180, B: 180, A: 255}
			switch item.Rarity {
			case 2:
				col = rl.Color{R: 80, G: 200, B: 255, A: 255}
			case 3:
				col = rl.Color{R: 255, G: 200, B: 50, A: 255}
			}
			rl.DrawText(item.Name, panelX, panelY+int32(row*16), 12, col)
		}
	}

	// Score and rooms
	rl.DrawText(fmt.Sprintf("Floor %d  |  Score: %d", game.RoomsCleared+1, game.Score),
		10, 10, 20, rl.White)

	// Dodge cooldown
	if p.DodgeCooldown > 0 {
		rl.DrawText(fmt.Sprintf("Dodge: %.1f", p.DodgeCooldown), barX, barY+24, 14, rl.Gray)
	} else {
		rl.DrawText("Dodge: Ready [Space/RMB]", barX, barY+24, 14, rl.Color{R: 100, G: 255, B: 100, A: 255})
	}

	// Class name
	classNames := []string{"Mage", "Warrior"}
	classColors := []rl.Color{
		{R: 100, G: 150, B: 255, A: 255},
		{R: 255, G: 100, B: 80, A: 255},
	}
	rl.DrawText(classNames[p.Class], barX, barY-36, 16, classColors[p.Class])

	// Stealth indicator
	stealthy := p.TimeSinceHit > 5.0 && p.FireTimer <= 0
	if stealthy {
		rl.DrawText("Stealth", barX+80, barY-36, 14, rl.Color{R: 100, G: 200, B: 180, A: 255})
	}

	// Melee cooldown (Warrior/Rogue)
	if p.Class != ClassMage {
		if p.MeleeCooldown > 0 {
			rl.DrawText(fmt.Sprintf("Attack: %.1f", p.MeleeCooldown), barX+140, barY+24, 14, rl.Gray)
		} else {
			rl.DrawText("Attack: Ready", barX+140, barY+24, 14, rl.Color{R: 255, G: 200, B: 100, A: 255})
		}
	}

	// Item pickup message
	if game.MessageTimer > 0 {
		alpha := uint8(255)
		if game.MessageTimer < 0.5 {
			alpha = uint8(game.MessageTimer / 0.5 * 255)
		}
		tw := rl.MeasureText(game.Message, 20)
		rl.DrawText(game.Message, (screenWidth-tw)/2, screenHeight/2-80, 20,
			rl.Color{R: 255, G: 220, B: 100, A: alpha})
	}

	// Floating text
	for i := range game.Floats {
		ft := &game.Floats[i]
		progress := 1.0 - ft.Timer/ft.MaxTime
		worldPos := rl.Vector3{X: ft.WorldX, Y: floorSurfaceY + tileUnit + float32(progress)*tileUnit*2, Z: ft.WorldZ}
		screenPos := rl.GetWorldToScreen(worldPos, camera)
		alpha := ft.Timer / ft.MaxTime
		if alpha > 1 {
			alpha = 1
		}
		col := rl.Color{R: ft.Color[0], G: ft.Color[1], B: ft.Color[2], A: uint8(alpha * 255)}
		tw := rl.MeasureText(ft.Text, ft.FontSize)
		rl.DrawText(ft.Text, int32(screenPos.X)-tw/2, int32(screenPos.Y), ft.FontSize, col)
	}

	rl.DrawFPS(screenWidth-90, screenHeight-25)
}

func drawTitle(selected PlayerClass) {
	rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.Color{R: 5, G: 5, B: 10, A: 255})
	title := "C R A W L Y"
	tw := rl.MeasureText(title, 48)
	rl.DrawText(title, (screenWidth-tw)/2, 80, 48, rl.Color{R: 255, G: 200, B: 60, A: 255})

	sub := "Twin-Stick Dungeon Crawler"
	sw := rl.MeasureText(sub, 20)
	rl.DrawText(sub, (screenWidth-sw)/2, 140, 20, rl.Color{R: 180, G: 180, B: 200, A: 255})

	// Class cards
	type classCard struct {
		name  string
		desc  string
		stats string
		col   rl.Color
	}
	cards := []classCard{
		{"Mage", "Ranged spellcaster", "HP: 6  SPD: 5  DMG: 3  |  Projectiles", rl.Color{R: 100, G: 150, B: 255, A: 255}},
		{"Warrior", "Heavy melee fighter", "HP: 10  SPD: 4  DMG: 5  |  Wide arc", rl.Color{R: 255, G: 100, B: 80, A: 255}},
	}

	cardW := int32(300)
	cardH := int32(120)
	gap := int32(20)
	totalW := cardW*int32(len(cards)) + gap*int32(len(cards)-1)
	startX := (screenWidth - totalW) / 2
	cardY := int32(220)

	for i, card := range cards {
		x := startX + int32(i)*(cardW+gap)
		isSelected := PlayerClass(i) == selected

		// Background
		bgCol := rl.Color{R: 20, G: 20, B: 30, A: 200}
		borderCol := rl.Color{R: 60, G: 60, B: 80, A: 255}
		if isSelected {
			bgCol = rl.Color{R: 30, G: 30, B: 50, A: 230}
			borderCol = card.col
		}
		rl.DrawRectangle(x, cardY, cardW, cardH, bgCol)
		rl.DrawRectangleLines(x, cardY, cardW, cardH, borderCol)

		// Name
		nameCol := rl.Color{R: 120, G: 120, B: 140, A: 255}
		if isSelected {
			nameCol = card.col
		}
		ntw := rl.MeasureText(card.name, 24)
		rl.DrawText(card.name, x+cardW/2-ntw/2, cardY+12, 24, nameCol)

		// Desc
		dtw := rl.MeasureText(card.desc, 14)
		rl.DrawText(card.desc, x+cardW/2-dtw/2, cardY+44, 14, rl.Color{R: 160, G: 160, B: 180, A: 255})

		// Stats
		stw := rl.MeasureText(card.stats, 12)
		rl.DrawText(card.stats, x+cardW/2-stw/2, cardY+70, 12, rl.Color{R: 140, G: 140, B: 160, A: 255})

		// Selected indicator
		if isSelected {
			pulse := uint8(float64(200) + math.Sin(float64(rl.GetTime())*3)*55)
			rl.DrawRectangleLines(x-2, cardY-2, cardW+4, cardH+4,
				rl.Color{R: card.col.R, G: card.col.G, B: card.col.B, A: pulse})
		}
	}

	// Controls
	controls := "A/D select  |  Space begin"
	cw := rl.MeasureText(controls, 18)
	pulse := uint8(float64(180) + math.Sin(float64(rl.GetTime())*3)*75)
	rl.DrawText(controls, (screenWidth-cw)/2, cardY+cardH+40, 18, rl.Color{R: pulse, G: pulse, B: pulse, A: 255})
}

func drawDeath(game *GameState) {
	rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.Color{R: 0, G: 0, B: 0, A: 180})
	title := "YOU HAVE FALLEN"
	tw := rl.MeasureText(title, 36)
	rl.DrawText(title, (screenWidth-tw)/2, screenHeight/2-60, 36, rl.Color{R: 255, G: 60, B: 60, A: 255})

	info := fmt.Sprintf("Reached Floor %d  |  Score: %d  |  Items: %d",
		game.RoomsCleared+1, game.Score, len(game.Player.Items))
	iw := rl.MeasureText(info, 20)
	rl.DrawText(info, (screenWidth-iw)/2, screenHeight/2, 20, rl.Color{R: 200, G: 200, B: 200, A: 255})

	restart := "[Space] Try Again"
	rw := rl.MeasureText(restart, 20)
	rl.DrawText(restart, (screenWidth-rw)/2, screenHeight/2+50, 20, rl.Gray)
}

func drawTransition(progress float32) {
	alpha := uint8(0)
	if progress < 0.5 {
		alpha = uint8(progress / 0.5 * 255)
	} else {
		alpha = uint8((1.0 - progress) / 0.5 * 255)
	}
	rl.DrawRectangle(0, 0, screenWidth, screenHeight, rl.Color{R: 0, G: 0, B: 0, A: alpha})
}
