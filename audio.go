package main

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type SFXGroup struct {
	Sounds []rl.Sound
	Volume float32
}

func loadSFXGroup(paths []string, volume float32) SFXGroup {
	g := SFXGroup{Volume: volume}
	for _, p := range paths {
		g.Sounds = append(g.Sounds, rl.LoadSound(p))
	}
	return g
}

func (g *SFXGroup) Play() {
	if len(g.Sounds) == 0 {
		return
	}
	s := g.Sounds[rand.Intn(len(g.Sounds))]
	rl.SetSoundVolume(s, g.Volume)
	rl.SetSoundPitch(s, 0.9+rand.Float32()*0.2)
	rl.PlaySound(s)
}

func (g *SFXGroup) Unload() {
	for _, s := range g.Sounds {
		rl.UnloadSound(s)
	}
}

type AudioSystem struct {
	MeleeSwing  SFXGroup
	MeleeHit    SFXGroup
	MageCast    SFXGroup
	EnemyHit    SFXGroup
	EnemyDeath  SFXGroup
	PlayerHit   SFXGroup
	ItemPickup  SFXGroup
	BlockParry  SFXGroup
	Dodge       SFXGroup
	UISelect    SFXGroup

	Ambient rl.Music
}

const audioBase = "assets/audio/"
const rpg = audioBase + "rpg-audio/Audio/"
const impact = audioBase + "impact-sounds/Audio/"
const ui = audioBase + "ui-audio/Audio/"

func LoadAudio() *AudioSystem {
	rl.InitAudioDevice()
	a := &AudioSystem{}

	a.MeleeSwing = loadSFXGroup([]string{
		rpg + "knifeSlice.ogg", rpg + "knifeSlice2.ogg", rpg + "chop.ogg",
	}, 0.5)

	a.MeleeHit = loadSFXGroup([]string{
		impact + "impactPunch_heavy_000.ogg", impact + "impactPunch_heavy_001.ogg",
		impact + "impactPunch_heavy_002.ogg", impact + "impactPunch_heavy_003.ogg",
		impact + "impactPunch_heavy_004.ogg",
	}, 0.6)

	a.MageCast = loadSFXGroup([]string{
		impact + "impactGlass_light_000.ogg", impact + "impactGlass_light_001.ogg",
		impact + "impactGlass_light_002.ogg", impact + "impactGlass_light_003.ogg",
		impact + "impactGlass_light_004.ogg",
	}, 0.4)

	a.EnemyHit = loadSFXGroup([]string{
		impact + "impactSoft_medium_000.ogg", impact + "impactSoft_medium_001.ogg",
		impact + "impactSoft_medium_002.ogg", impact + "impactSoft_medium_003.ogg",
		impact + "impactSoft_medium_004.ogg",
	}, 0.5)

	a.EnemyDeath = loadSFXGroup([]string{
		impact + "impactSoft_heavy_000.ogg", impact + "impactSoft_heavy_001.ogg",
		impact + "impactSoft_heavy_002.ogg", impact + "impactSoft_heavy_003.ogg",
		impact + "impactSoft_heavy_004.ogg",
	}, 0.6)

	a.PlayerHit = loadSFXGroup([]string{
		impact + "impactMetal_medium_000.ogg", impact + "impactMetal_medium_001.ogg",
		impact + "impactMetal_medium_002.ogg", impact + "impactMetal_medium_003.ogg",
		impact + "impactMetal_medium_004.ogg",
	}, 0.7)

	a.ItemPickup = loadSFXGroup([]string{
		rpg + "handleCoins.ogg", rpg + "handleCoins2.ogg", rpg + "clothBelt.ogg",
	}, 0.5)

	a.BlockParry = loadSFXGroup([]string{
		impact + "impactMetal_heavy_000.ogg", impact + "impactMetal_heavy_001.ogg",
		impact + "impactMetal_heavy_002.ogg", impact + "impactMetal_heavy_003.ogg",
		impact + "impactMetal_heavy_004.ogg",
	}, 0.6)

	a.Dodge = loadSFXGroup([]string{
		rpg + "cloth1.ogg", rpg + "cloth2.ogg", rpg + "cloth3.ogg", rpg + "cloth4.ogg",
	}, 0.4)

	a.UISelect = loadSFXGroup([]string{
		ui + "click1.ogg", ui + "click2.ogg", ui + "click3.ogg",
	}, 0.5)

	a.Ambient = rl.LoadMusicStream(audioBase + "ambient/dark_cavern_loop.ogg")
	a.Ambient.Looping = true
	rl.SetMusicVolume(a.Ambient, 0.3)
	rl.PlayMusicStream(a.Ambient)

	return a
}

func (a *AudioSystem) Update() {
	rl.UpdateMusicStream(a.Ambient)
}

func (a *AudioSystem) Unload() {
	rl.UnloadMusicStream(a.Ambient)
	a.MeleeSwing.Unload()
	a.MeleeHit.Unload()
	a.MageCast.Unload()
	a.EnemyHit.Unload()
	a.EnemyDeath.Unload()
	a.PlayerHit.Unload()
	a.ItemPickup.Unload()
	a.BlockParry.Unload()
	a.Dodge.Unload()
	a.UISelect.Unload()
	rl.CloseAudioDevice()
}
