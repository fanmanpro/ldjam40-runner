package main

import (
	"github.com/autovelop/playthos"
	"github.com/autovelop/playthos/animation"
	// "github.com/autovelop/playthos/audio"
	"github.com/autovelop/playthos/collision"
	"github.com/autovelop/playthos/keyboard"
	"github.com/autovelop/playthos/physics"
	"github.com/autovelop/playthos/render"
	"github.com/autovelop/playthos/scripting"
	"github.com/autovelop/playthos/std"

	// Linux platform only
	// _ "github.com/autovelop/playthos/glfw"
	// _ "github.com/autovelop/playthos/glfw/keyboard"
	// _ "github.com/autovelop/playthos/openal"
	// _ "github.com/autovelop/playthos/opengl"
	// Web platform only
	_ "github.com/autovelop/playthos/platforms/web/audio"
	_ "github.com/autovelop/playthos/platforms/web/keyboard"
	_ "github.com/autovelop/playthos/webgl"

	"log"
	"math/rand"
)

var game *engine.Engine
var player *engine.Entity
var player_rb *physics.RigidBody
var player_position std.Vector3
var camera_position std.Vector3
var camera_lookat std.Vector3
var can_jump = false
var spike_list []*engine.Entity
var rocket_list []*engine.Entity
var speed *float32
var ended = false
var spawner_script *scripting.Script

var bg0_anim *animation.AnimationClip
var bg1_anim *animation.AnimationClip
var bg2_anim *animation.AnimationClip

var level float32
var level_progress float32
var difficulty_modifier float32
var x float32
var min_distance float32
var double_spawn_chance int
var level_progression float32

var idleAnimation *animation.AnimationClip

func main() {
	engine.RegisterAsset("assets/player.png")
	engine.RegisterAsset("assets/rocket.png")
	engine.RegisterAsset("assets/spike.png")
	engine.RegisterAsset("assets/instructions.png")
	engine.RegisterAsset("assets/backgrounds0.png")
	engine.RegisterAsset("assets/backgrounds1.png")
	engine.RegisterAsset("assets/backgrounds2.png")

	game = engine.New("LDJAM40-Runner", &engine.Settings{
		false,
		1024,
		768,
		false,
	})
	kb := game.Listener(&keyboard.Keyboard{})

	kb.On(keyboard.KeyEscape, func(action ...int) {
		switch action[0] {
		case keyboard.ActionRelease:
			game.Stop()
		}
	})

	// mu := game.NewEntity()
	// clip := audio.NewClip()
	// clip.LoadClip("assets/MoodyLoop.wav")
	// music := audio.NewSound()
	// music.Set(clip)
	// mu.AddComponent(music)

	// src := audio.NewSource()
	// src.Set(&std.Vector3{0, 0, 0}, false, true)
	// src.PlaySound(music)
	// mu.AddComponent(src)

	cam := game.NewEntity()
	t := std.NewTransform()
	camera_position = std.Vector3{0, 0, 100}
	camera_lookat = std.Vector3{0, 0, 0}
	t.Set(
		&camera_position,      // POSITION
		&camera_lookat,        // CENTER
		&std.Vector3{0, 1, 0}, // UP
	)
	cam.AddComponent(t)

	camera := render.NewCamera()
	cameraSize := float32(4)
	camera.Set(&cameraSize, &std.Color{0.1, 0.1, 0.1, 0})
	camera.SetTransform(t)

	cam.AddComponent(camera)

	// Follow Cam
	follow_cam := scripting.NewScript()
	follow_cam.OnUpdate(func() {
		camera_position.Add(&std.Vector3{player_position.X - camera_position.X + 60, 0, 0})
		camera_lookat.Add(&std.Vector3{player_position.X - camera_lookat.X + 60, 0, 0})
	})
	cam.AddComponent(follow_cam)

	/*
	 Create Player
	*/
	player = game.NewEntity()
	transform := std.NewTransform()
	player_position = std.Vector3{0, -25 - 54, 80}
	transform.Set(
		&player_position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{12, 12, 1})
	player.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	player.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{1, 1, 1, 1},
	)

	player_image := render.NewImage()
	player_image.LoadImage("assets/player.png")
	player_sprite := render.NewTexture(player_image)
	player_sprite.SetSize(100, 100)
	player_offset := std.Vector2{0, 0}
	player_sprite.SetOffset(&player_offset)
	material.SetTexture(player_sprite)

	player.AddComponent(material)

	player_rb = physics.NewRigidBody()
	player_rb.SetMass(2.5)

	player_rb.SetVelocity(0.4, 0, 0)
	player.AddComponent(player_rb)

	col := collision.NewCollider()
	col.Set(&player_position, &std.Vector2{0, 0}, &std.Vector2{12, 12})
	player.AddComponent(col)

	idleAnimation = animation.NewClip(1, 100, &player_offset)
	idleAnimation.AddKeyFrame(0, 49, &std.Vector2{0, 0})
	idleAnimation.AddKeyFrame(50, 49, &std.Vector2{1, 0})
	idleAnimation.AddKeyFrame(100, 49, &std.Vector2{2, 0})
	player.AddComponent(idleAnimation)

	kb.On(keyboard.KeySpace, func(action ...int) {
		switch action[0] {
		case keyboard.ActionPress:
			if ended {
				restart()
			} else {
				if can_jump {
					can_jump = false
					player_rb.SetVelocity(player_rb.Velocity().X, 1, 0)
					player_rb.SetMass(2.5)
				}
			}
		}
	})

	ground()
	instructions()
	bg0_anim = background("assets/backgrounds0.png", 6000, 0)
	bg1_anim = background("assets/backgrounds1.png", 4000, 1)
	bg2_anim = background("assets/backgrounds2.png", 2000, 2)

	// Spawner
	spike_list = make([]*engine.Entity, 0)
	rocket_list = make([]*engine.Entity, 0)
	spawner := game.NewEntity()
	spawner_script = scripting.NewScript()
	restart()
	rocket(x + 65)
	spawner_script.OnUpdate(func() {
		level_progress = player_position.X - float32(level*level_progression)
		if level_progress > level_progression {
			level_progress = 0
			level_progression++
			level++
			distance := float32(-100)
			if difficulty_modifier > 0.01 {
				distance = float32(rand.Intn(int(200 * difficulty_modifier)))
				difficulty_modifier *= 0.95
			}
			x += min_distance + distance
			spike(x)
			if rand.Intn(int(level_progression)) >= double_spawn_chance {
				spike(x + 12 + float32(rand.Intn(10)))
			} else {
				if rand.Intn(int(100)) >= 50 {
					rocket(x + 50 + float32(rand.Intn(80)))
					if rand.Intn(int(100)) >= 50 {
						rocket(x + 65 + float32(rand.Intn(30)))
					}
				}
			}
			if double_spawn_chance > 0 {
				double_spawn_chance -= 2
			} else {
				double_spawn_chance = 0
				if rand.Intn(int(100)) >= 50 {
					rocket(x + 50 + float32(rand.Intn(80)))
					if rand.Intn(int(100)) >= 50 {
						rocket(x + 65 + float32(rand.Intn(30)))
					}
				}
			}
			x = (level + 2) * float32(level_progression)

			// increase speed
			if player_rb.Velocity().X < 1 {
				player_rb.SetVelocity(player_rb.Velocity().X+0.007, player_rb.Velocity().Y, 0)
			}
		}
	})
	spawner.AddComponent(spawner_script)

	game.Start()
}

func restart() {
	for _, spike := range spike_list {
		game.DeleteEntity(spike)
	}
	for _, rocket := range rocket_list {
		game.DeleteEntity(rocket)
	}
	spike_list = make([]*engine.Entity, 0)
	rocket_list = make([]*engine.Entity, 0)
	level = float32(0)
	level_progress = float32(0)
	spike(400)
	difficulty_modifier = float32(1)
	x = (level + 2) * float32(200)
	min_distance = float32(30)
	double_spawn_chance = 190
	level_progression = float32(200)
	player_position.Add(&std.Vector3{-player_position.X, -25 - 58.5 - player_position.Y, 0})
	player_rb.SetVelocity(0.4, 0, 0)
	player_rb.SetActive(true)
	ended = false
	bg0_anim.Start()
	bg1_anim.Start()
	bg2_anim.Start()
	idleAnimation.Start()
}

func rocket(x float32) {
	// Ground
	rocket := game.NewEntity()
	transform := std.NewTransform()
	position := std.Vector3{x, 12 - 54, 3}
	transform.Set(
		&position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{20, 10, 1})
	rocket.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	rocket.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{1, 1, 1, 1},
	)
	rocket_image := render.NewImage()
	rocket_image.LoadImage("assets/rocket.png")
	rocket_sprite := render.NewTexture(rocket_image)
	rocket_sprite.SetSize(150, 50)
	rocket_offset := std.Vector2{0, 0}
	rocket_sprite.SetOffset(&rocket_offset)
	material.SetTexture(rocket_sprite)
	rocket.AddComponent(material)

	col := collision.NewCollider()
	col.Set(&position, &std.Vector2{0, 0}, &std.Vector2{20, 10})
	col.OnHit(func(other *collision.Collider) {
		if other.Entity() == player {
			ended = true
			log.Printf("Your Score: %v", player_position.X)
			bg0_anim.Stop()
			bg1_anim.Stop()
			bg2_anim.Stop()
			idleAnimation.Stop()
			player_rb.SetActive(false)
		}
	})
	rocket.AddComponent(col)

	idleAnimation := animation.NewClip(1, 200, &rocket_offset)
	idleAnimation.AddKeyFrame(0, 49, &std.Vector2{0, 0})
	idleAnimation.AddKeyFrame(50, 49, &std.Vector2{1, 0})
	idleAnimation.AddKeyFrame(100, 49, &std.Vector2{2, 0})
	idleAnimation.AddKeyFrame(150, 49, &std.Vector2{3, 0})
	idleAnimation.AddKeyFrame(200, 49, &std.Vector2{4, 0})
	rocket.AddComponent(idleAnimation)

	rocket_list = append(rocket_list, rocket)
	if len(rocket_list) > 6 {
		rocket_list[0].SetActive(false)
	}
}

func spike(x float32) {
	// Ground
	spike := game.NewEntity()
	transform := std.NewTransform()
	position := std.Vector3{x, -28 - 54, 3}
	transform.Set(
		&position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{10, 16, 1})
	spike.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	spike.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{1, 1, 1, 1},
	)
	spike_image := render.NewImage()
	spike_image.LoadImage("assets/spike.png")
	spike_sprite := render.NewTexture(spike_image)
	spike_sprite.SetSize(100, 160)
	spike_offset := std.Vector2{0, 0}
	spike_sprite.SetOffset(&spike_offset)
	material.SetTexture(spike_sprite)

	spike.AddComponent(material)

	col := collision.NewCollider()
	col.Set(&position, &std.Vector2{0, 0}, &std.Vector2{10, 16})
	col.OnHit(func(other *collision.Collider) {
		if other.Entity() == player && !ended {
			ended = true
			log.Printf("Your Score: %v", player_position.X)
			bg0_anim.Stop()
			bg1_anim.Stop()
			bg2_anim.Stop()
			idleAnimation.Stop()
			player_rb.SetActive(false)
		}
	})
	spike.AddComponent(col)

	spike_list = append(spike_list, spike)
	if len(spike_list) > 6 {
		spike_list[0].SetActive(false)
	}
}

func instructions() {
	// Ground
	instructions := game.NewEntity()
	transform := std.NewTransform()
	position := std.Vector3{0, 60, 3}
	transform.Set(
		&position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{64, 32, 1})
	instructions.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	instructions.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{1, 1, 1, 1},
	)
	instructions_image := render.NewImage()
	instructions_image.LoadImage("assets/instructions.png")
	instructions_sprite := render.NewTexture(instructions_image)
	instructions_sprite.SetSize(64, 32)
	instructions_offset := std.Vector2{0, 0}
	instructions_sprite.SetOffset(&instructions_offset)
	material.SetTexture(instructions_sprite)
	instructions.AddComponent(material)

	// Follow Cam
	follow_static := scripting.NewScript()
	follow_static.OnUpdate(func() {
		position.Add(&std.Vector3{player_position.X - position.X + 60, 0, 0})
	})
	instructions.AddComponent(follow_static)
}

func ground() {
	// Ground
	ground := game.NewEntity()
	transform := std.NewTransform()
	position := std.Vector3{0, -94, 3}
	transform.Set(
		&position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{999999, 10, 1})
	ground.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	ground.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{0.05, 0.05, 0.1, 1},
	)
	ground.AddComponent(material)

	col := collision.NewCollider()
	col.Set(&position, &std.Vector2{0, 0}, &std.Vector2{999999, 10})
	col.OnHit(func(other *collision.Collider) {
		if !can_jump {
			can_jump = true
			player_rb.SetVelocity(player_rb.Velocity().X, 0, 0)
			player_rb.SetMass(0)
			player_position.Add(&std.Vector3{0, 0.4, 0})
		}
	})
	ground.AddComponent(col)
}

func background(img string, speed float64, depth float32) *animation.AnimationClip {
	background := game.NewEntity()
	transform := std.NewTransform()
	position := std.Vector3{0, 0, depth}
	transform.Set(
		&position,
		&std.Vector3{0, 0, 0},
		&std.Vector3{320 * 0.8, 240 * 0.8, 1})
	background.AddComponent(transform)

	quad := render.NewMesh()
	quad.Set(std.QuadMesh)
	background.AddComponent(quad)

	material := render.NewMaterial()
	material.SetColor(
		&std.Color{1, 1, 1, 1},
	)
	bg_image := render.NewImage()
	bg_image.LoadImage(img)
	bg_sprite := render.NewTexture(bg_image)
	bg_sprite.SetSize(1024, 768)
	bg_offset := std.Vector2{1, 0}
	bg_sprite.SetOffset(&bg_offset)
	material.SetTexture(bg_sprite)

	background.AddComponent(material)

	bg_anim := animation.NewClip(1, speed, &bg_offset)
	bg_anim.AddKeyFrame(0, 0, &std.Vector2{0, 0})
	bg_anim.AddKeyFrame(speed, 0, &std.Vector2{1, 0})
	background.AddComponent(bg_anim)

	// Follow Cam
	follow_static := scripting.NewScript()
	follow_static.OnUpdate(func() {
		position.Add(&std.Vector3{player_position.X - position.X + 60, 0, 0})
	})
	background.AddComponent(follow_static)

	return bg_anim
}
