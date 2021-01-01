// Package input exposes the two input callbacks Poll and State needed by the
// libretro implementation. It uses GLFW to access keyboard and joypad, and
// takes care of binding and auto configuring joypads.
package input

import (
	deepcopy "github.com/barkimedes/go-deepcopy"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/libretro/ludo/libretro"
	ntf "github.com/libretro/ludo/notifications"
	"github.com/libretro/ludo/settings"
	"github.com/libretro/ludo/state"
	"github.com/libretro/ludo/video"
)

// MaxPlayers is the maximum number of players to poll input for
const MaxPlayers = 5

const maxFrames = 60
const localPlayerPort = 0
const remotePlayerPort = 1

var polled = inputState{}
var buffers = []inputState{}

type joybinds map[bind]uint32

const btn = 0
const axis = 1

type bind struct {
	kind      uint32
	index     uint32
	direction float32
	threshold float32
}

type playerState [ActionLast]bool
type inputState [MaxPlayers]playerState

// Input state for all the players
var (
	NewState inputState // input state for the current frame
	OldState inputState // input state for the previous frame
	Released inputState // keys just released during this frame
	Pressed  inputState // keys just pressed during this frame
)

// Hot keys
const (
	// ActionMenuToggle toggles the menu UI
	ActionMenuToggle uint32 = libretro.DeviceIDJoypadR3 + 1
	// ActionFullscreenToggle switches between fullscreen and windowed mode
	ActionFullscreenToggle uint32 = libretro.DeviceIDJoypadR3 + 2
	// ActionShouldClose will cause the program to shutdown
	ActionShouldClose uint32 = libretro.DeviceIDJoypadR3 + 3
	// ActionFastForwardToggle will run the core as fast as possible
	ActionFastForwardToggle uint32 = libretro.DeviceIDJoypadR3 + 4
	// ActionLast is used for iterating
	ActionLast uint32 = libretro.DeviceIDJoypadR3 + 5
)

func index(offset int) int {
	tick := state.Global.Tick
	tick += offset
	return (maxFrames + tick) % maxFrames
}

func Serialize() interface{} {
	copy, err := deepcopy.Anything(buffers)
	if err != nil {
		panic(err)
	}
	return copy
}

func Unserialize(st interface{}) {
	copy, err := deepcopy.Anything(st)
	if err != nil {
		panic(err)
	}
	buffers = copy.([]inputState)
}

func getState(port uint, tick int) playerState {
	frame := (maxFrames + tick) % maxFrames
	st := buffers[port][frame]
	return st
}

func getLatest(port uint) playerState {
	return polled[port]
}

func currentState(port uint) playerState {
	return getState(port, state.Global.Tick)
}

func setState(port uint, st playerState) {
	copy, err := deepcopy.Anything(st)
	if err != nil {
		panic(err)
	}
	buffers[port][index(0)] = copy.(playerState)
}

func initializeBuffer(port uint) {
	for i := 0; i < maxFrames; i++ {
		buffers[port][i] = playerState{}
	}
}

// joystickCallback is triggered when a joypad is plugged.
func joystickCallback(joy glfw.Joystick, event glfw.PeripheralEvent) {
	switch event {
	case glfw.Connected:
		if HasBinding(joy) {
			ntf.DisplayAndLog(ntf.Info, "Input", "Joystick #%d plugged: %s.", joy, glfw.Joystick.GetName(joy))
		} else {
			ntf.DisplayAndLog(ntf.Warning, "Input", "Joystick #%d plugged: %s but not configured.", joy, glfw.Joystick.GetName(joy))
		}
	case glfw.Disconnected:
		ntf.DisplayAndLog(ntf.Info, "Input", "Joystick #%d unplugged.", joy)
	default:
		ntf.DisplayAndLog(ntf.Warning, "Input", "Joystick #%d unhandled event: %d.", joy, event)
	}
}

var vid *video.Video

// Init initializes the input package
func Init(v *video.Video) {
	vid = v
	glfw.SetJoystickCallback(joystickCallback)
}

// pollJoypads process joypads of all players
func pollJoypads() {
	for p := range polled {
		buttonState := glfw.Joystick.GetButtons(glfw.Joystick(p))
		axisState := glfw.Joystick.GetAxes(glfw.Joystick(p))
		name := glfw.Joystick.GetName(glfw.Joystick(p))
		jb := joyBinds[name]
		if len(buttonState) > 0 {
			for k, v := range jb {
				switch k.kind {
				case btn:
					if int(k.index) < len(buttonState) &&
						glfw.Action(buttonState[k.index]) == glfw.Press {
						polled[p][v] = true
					}
				case axis:
					if int(k.index) < len(axisState) &&
						k.direction*axisState[k.index] > k.threshold*k.direction {
						polled[p][v] = true
					}
				}

				if !settings.Current.MapAxisToDPad {
					continue
				}
				switch {
				case axisState[0] < -0.5:
					polled[p][libretro.DeviceIDJoypadLeft] = true
				case axisState[0] > 0.5:
					polled[p][libretro.DeviceIDJoypadRight] = true
				}
				switch {
				case axisState[1] > 0.5:
					polled[p][libretro.DeviceIDJoypadDown] = true
				case axisState[1] < -0.5:
					polled[p][libretro.DeviceIDJoypadUp] = true
				}
			}
		}
	}
}

// pollKeyboard processes keyboard keys
func pollKeyboard() {
	for k, v := range keyBinds {
		if vid.Window.GetKey(k) == glfw.Press {
			polled[localPlayerPort][v] = true
		}
	}
}

// Compute the keys pressed or released during this frame
func getPressedReleased(new inputState, old inputState) (inputState, inputState) {
	for p := range new {
		for k := range new[p] {
			Pressed[p][k] = new[p][k] && !old[p][k]
			Released[p][k] = !new[p][k] && old[p][k]
		}
	}
	return Pressed, Released
}

// Poll calculates the input state. It is meant to be called for each frame.
func Poll() {
	polled = inputState{}
	pollKeyboard()
	pollJoypads()
	NewState = polled

	Pressed, Released = getPressedReleased(NewState, OldState)

	// Store the old input state for comparisions
	OldState = NewState
}

// State is a callback passed to core.SetInputState
// It returns 1 if the button corresponding to the parameters is pressed
func State(port uint, device uint32, index uint, id uint) int16 {
	if id >= 255 || index > 0 || port >= MaxPlayers || device&libretro.DeviceJoypad != 1 || id > uint(libretro.DeviceIDJoypadR3) {
		return 0
	}

	if getLatest(port)[id] {
		return 1
	}
	return 0
}

// HasBinding returns true if the joystick has an autoconfig binding
func HasBinding(joy glfw.Joystick) bool {
	name := glfw.Joystick.GetName(joy)
	_, ok := joyBinds[name]
	return ok
}
