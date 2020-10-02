package main

import (
	"flag"
	"github.com/faiface/mainthread"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/wieku/danser-go/app/audio"
	"github.com/wieku/danser-go/app/beatmap"
	"github.com/wieku/danser-go/app/bmath"
	"github.com/wieku/danser-go/app/dance"
	"github.com/wieku/danser-go/app/database"
	"github.com/wieku/danser-go/app/discord"
	"github.com/wieku/danser-go/app/graphics/font"
	"github.com/wieku/danser-go/app/input"
	"github.com/wieku/danser-go/app/settings"
	"github.com/wieku/danser-go/app/states"
	"github.com/wieku/danser-go/app/utils"
	"github.com/wieku/danser-go/build"
	"github.com/wieku/danser-go/framework/bass"
	"github.com/wieku/danser-go/framework/frame"
	"github.com/wieku/danser-go/framework/graphics/blend"
	"github.com/wieku/danser-go/framework/graphics/sprite"
	"github.com/wieku/danser-go/framework/graphics/viewport"
	"github.com/wieku/danser-go/framework/math/vector"
	"github.com/wieku/danser-go/framework/statistic"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

var player *states.Player
var pressed = false
var pressedM = false
var pressedP = false

func run() {
	var win *glfw.Window
	var limiter *frame.Limiter

	mainthread.Call(func() {

		artist := flag.String("artist", "", "")
		artistS := flag.String("a", "", "")

		title := flag.String("title", "", "")
		titleS := flag.String("t", "", "")

		difficulty := flag.String("difficulty", "", "")
		difficultyS := flag.String("d", "", "")

		creator := flag.String("creator", "", "")
		creatorS := flag.String("c", "", "")

		settingsVersion := flag.Int("settings", 0, "")
		cursors := flag.Int("cursors", 1, "")
		tag := flag.Int("tag", 1, "")
		knockout := flag.Bool("knockout", false, "")
		speed := flag.Float64("speed", 1.0, "")
		pitch := flag.Float64("pitch", 1.0, "")
		mover := flag.String("mover", "flower", "")
		debug := flag.Bool("debug", false, "")

		play := flag.Bool("play", false, "")

		flag.Parse()

		closeAfterSettingsLoad := false

		if (*artist + *title + *difficulty + *creator + *artistS + *titleS + *difficultyS + *creatorS) == "" {
			log.Println("No beatmap specified, closing...")
			closeAfterSettingsLoad = true
		}

		settings.DEBUG = *debug
		settings.KNOCKOUT = *knockout
		settings.PLAY = *play
		settings.DIVIDES = *cursors
		settings.TAG = *tag
		settings.SPEED = *speed
		settings.PITCH = *pitch
		_ = mover
		dance.SetMover(*mover)

		newSettings := settings.LoadSettings(*settingsVersion)

		player = nil
		var beatMap *beatmap.BeatMap = nil

		if !closeAfterSettingsLoad {
			database.Init()
			beatmaps := database.LoadBeatmaps()

			for _, b := range beatmaps {
				if (*artist == "" || *artist == b.Artist) && (*artistS == "" || *artistS == b.Artist) &&
					(*title == "" || *title == b.Name) && (*titleS == "" || *titleS == b.Name) &&
					(*difficulty == "" || *difficulty == b.Difficulty) && (*difficultyS == "" || *difficultyS == b.Difficulty) &&
					(*creator == "" || *creator == b.Creator) && (*creatorS == "" || *creatorS == b.Creator) {
					beatMap = b
					beatMap.UpdatePlayStats()
					database.UpdatePlayStats(beatMap)
					break
				}
			}

			if beatMap == nil {
				log.Println("Beatmap not found, closing...")
				closeAfterSettingsLoad = true
			} else {
				discord.Connect()
			}
		}

		glfw.Init()
		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, glfw.False)
		glfw.WindowHint(glfw.Samples, int(settings.Graphics.MSAA))

		var err error

		monitor := glfw.GetPrimaryMonitor()
		mWidth, mHeight := monitor.GetVideoMode().Width, monitor.GetVideoMode().Height

		if newSettings {
			settings.Graphics.SetDefaults(int64(mWidth), int64(mHeight))
			settings.Save()
		}

		if closeAfterSettingsLoad {
			os.Exit(0)
		}

		if settings.Graphics.Fullscreen {
			glfw.WindowHint(glfw.RedBits, monitor.GetVideoMode().RedBits)
			glfw.WindowHint(glfw.GreenBits, monitor.GetVideoMode().GreenBits)
			glfw.WindowHint(glfw.BlueBits, monitor.GetVideoMode().BlueBits)
			glfw.WindowHint(glfw.RefreshRate, monitor.GetVideoMode().RefreshRate)
			//glfw.WindowHint(glfw.Decorated, glfw.False)
			win, err = glfw.CreateWindow(int(settings.Graphics.Width), int(settings.Graphics.Height), "danser", monitor, nil)
		} else {
			win, err = glfw.CreateWindow(int(settings.Graphics.WindowWidth), int(settings.Graphics.WindowHeight), "danser", nil, nil)
		}

		if err != nil {
			panic(err)
		}

		win.SetTitle("danser " + build.VERSION + " - " + beatMap.Artist + " - " + beatMap.Name + " [" + beatMap.Difficulty + "]")
		input.Win = win
		icon, _ := utils.LoadImageN("assets/textures/dansercoin.png")
		icon2, _ := utils.LoadImageN("assets/textures/dansercoin48.png")
		icon3, _ := utils.LoadImageN("assets/textures/dansercoin24.png")
		icon4, _ := utils.LoadImageN("assets/textures/dansercoin16.png")
		win.SetIcon([]image.Image{icon, icon2, icon3, icon4})

		win.MakeContextCurrent()
		log.Println("GLFW initialized!")
		gl.Init()
		gl.Enable(gl.BLEND)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		batch := sprite.NewSpriteBatch()
		batch.Begin()
		batch.SetColor(1, 1, 1, 1)
		camera := bmath.NewCamera()
		camera.SetViewport(int(settings.Graphics.GetWidth()), int(settings.Graphics.GetHeight()), false)
		camera.SetOrigin(vector.NewVec2d(settings.Graphics.GetWidthF()/2, settings.Graphics.GetHeightF()/2))
		camera.Update()
		batch.SetCamera(camera.GetProjectionView())

		file, _ := os.Open("assets/fonts/Exo2-Bold.ttf")
		font := font.LoadFont(file)
		file.Close()

		font.Draw(batch, 0, 10, 32, "Loading...")

		batch.End()
		win.SwapBuffers()
		glfw.PollEvents()

		glfw.SwapInterval(0)
		if settings.Graphics.VSync {
			glfw.SwapInterval(1)
		}

		bass.Init()
		audio.LoadSamples()

		beatmap.ParseObjects(beatMap)
		beatMap.LoadCustomSamples()
		player = states.NewPlayer(beatMap)
		limiter = frame.NewLimiter(int(settings.Graphics.FPSCap))
	})

	for !win.ShouldClose() {
		mainthread.Call(func() {
			statistic.Reset()
			glfw.PollEvents()

			if settings.Graphics.MSAA > 0 {
				gl.Enable(gl.MULTISAMPLE)
			}

			gl.Enable(gl.SCISSOR_TEST)
			gl.Disable(gl.DITHER)

			viewport.Push(int(settings.Graphics.GetWidth()), int(settings.Graphics.GetHeight()))

			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT)

			if player != nil {
				player.Draw(0)
			}

			if win.GetKey(glfw.KeyEscape) == glfw.Press {
				win.SetShouldClose(true)
			}

			if win.GetKey(glfw.KeyF2) == glfw.Press {

				if !pressed {
					utils.MakeScreenshot(*win)
				}

				pressed = true
			}

			if win.GetKey(glfw.KeyF2) == glfw.Release {
				pressed = false
			}

			if win.GetKey(glfw.KeyMinus) == glfw.Press {

				if !pressedM {
					if settings.DIVIDES > 1 {
						settings.DIVIDES -= 1
					}
				}

				pressedM = true
			}

			if win.GetKey(glfw.KeyMinus) == glfw.Release {
				pressedM = false
			}

			if win.GetKey(glfw.KeyEqual) == glfw.Press {

				if !pressedP {
					settings.DIVIDES += 1
				}

				pressedP = true
			}

			if win.GetKey(glfw.KeyEqual) == glfw.Release {
				pressedP = false
			}

			win.SwapBuffers()

			if !settings.Graphics.VSync {
				limiter.Sync()
			}

			blend.ClearStack()
			viewport.ClearStack()
		})
	}
}

func setWorkingDirectory() {
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	if exec, err = filepath.EvalSymlinks(exec); err != nil {
		panic(err)
	}

	if err = os.Chdir(filepath.Dir(exec)); err != nil {
		panic(err)
	}
}

func main() {
	defer discord.Disconnect()
	setWorkingDirectory()
	runtime.GOMAXPROCS(runtime.NumCPU())
	mainthread.CallQueueCap = 100000
	mainthread.Run(run)
}
