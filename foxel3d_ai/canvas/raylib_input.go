// 该文件完全ai自主控制
package canvas

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var raylib_keys = map[Key]int32{
	KEY_A: rl.KeyA, KEY_B: rl.KeyB, KEY_C: rl.KeyC, KEY_D: rl.KeyD, KEY_E: rl.KeyE, KEY_F: rl.KeyF,
	KEY_G: rl.KeyG, KEY_H: rl.KeyH, KEY_I: rl.KeyI, KEY_J: rl.KeyJ, KEY_K: rl.KeyK, KEY_L: rl.KeyL,
	KEY_M: rl.KeyM, KEY_N: rl.KeyN, KEY_O: rl.KeyO, KEY_P: rl.KeyP, KEY_Q: rl.KeyQ, KEY_R: rl.KeyR,
	KEY_S: rl.KeyS, KEY_T: rl.KeyT, KEY_U: rl.KeyU, KEY_V: rl.KeyV, KEY_W: rl.KeyW, KEY_X: rl.KeyX,
	KEY_Y: rl.KeyY, KEY_Z: rl.KeyZ,
	KEY_0: rl.KeyZero, KEY_1: rl.KeyOne, KEY_2: rl.KeyTwo, KEY_3: rl.KeyThree, KEY_4: rl.KeyFour,
	KEY_5: rl.KeyFive, KEY_6: rl.KeySix, KEY_7: rl.KeySeven, KEY_8: rl.KeyEight, KEY_9: rl.KeyNine,
	KEY_SPACE: rl.KeySpace, KEY_ENTER: rl.KeyEnter, KEY_ESCAPE: rl.KeyEscape, KEY_TAB: rl.KeyTab,
	KEY_BACKSPACE:  rl.KeyBackspace,
	KEY_LEFT_SHIFT: rl.KeyLeftShift, KEY_RIGHT_SHIFT: rl.KeyRightShift,
	KEY_LEFT_CTRL: rl.KeyLeftControl, KEY_RIGHT_CTRL: rl.KeyRightControl,
	KEY_LEFT_ALT: rl.KeyLeftAlt, KEY_RIGHT_ALT: rl.KeyRightAlt,
	KEY_UP: rl.KeyUp, KEY_DOWN: rl.KeyDown, KEY_LEFT: rl.KeyLeft, KEY_RIGHT: rl.KeyRight,
	KEY_F1: rl.KeyF1, KEY_F2: rl.KeyF2, KEY_F3: rl.KeyF3, KEY_F4: rl.KeyF4, KEY_F5: rl.KeyF5,
	KEY_F6: rl.KeyF6, KEY_F7: rl.KeyF7, KEY_F8: rl.KeyF8, KEY_F9: rl.KeyF9, KEY_F10: rl.KeyF10,
	KEY_F11: rl.KeyF11, KEY_F12: rl.KeyF12,
}

var raylib_mouse = map[Key]rl.MouseButton{
	MOUSE_LEFT:   rl.MouseButtonLeft,
	MOUSE_RIGHT:  rl.MouseButtonRight,
	MOUSE_MIDDLE: rl.MouseButtonMiddle,
}

var raylib_keys_reverse = func() map[int32]Key {
	m := map[int32]Key{}
	for k, code := range raylib_keys {
		m[code] = k
	}
	return m
}()

// 上一帧鼠标的状态，用来丢掉突然很大的移动量
type mouse_state struct {
	// 窗口有焦点，鼠标在窗口里或者被锁定
	tracking bool
	locked   bool
}

func (r *Raylib) Get_input() Input {
	return r.input
}

func (r *Raylib) Set_cursor_locked(locked bool) {
	if locked && !rl.IsCursorHidden() {
		rl.DisableCursor()
	} else if !locked && rl.IsCursorHidden() {
		rl.EnableCursor()
	}
}

func (r *Raylib) Is_cursor_locked() bool {
	return rl.IsCursorHidden()
}

// 每帧画完后调用一次（EndDrawing里raylib刚处理完事件）
func (r *Raylib) read_input() Input {
	// 鼠标刚进入窗口、窗口刚获得焦点、刚锁定或解锁鼠标的那一帧，raylib是从旧位置算移动量的，会突然很大，丢掉这一帧的
	mouse := mouse_state{rl.IsWindowFocused() && (rl.IsCursorHidden() || rl.IsCursorOnScreen()), rl.IsCursorHidden()}
	mouse_ok := mouse.tracking && mouse == r.mouse
	r.mouse = mouse

	in := Input{Held: map[Key]bool{}, Pressed: map[Key]bool{}}
	if !rl.IsWindowFocused() {
		return in
	}
	// raylib每帧处理一次事件：两帧之间按下又松开的键IsKeyDown看不到，但会留在GetKeyPressed的队列里
	for code := rl.GetKeyPressed(); code != 0; code = rl.GetKeyPressed() {
		if k, ok := raylib_keys_reverse[code]; ok {
			in.Pressed[k] = true
			in.Held[k] = true
		}
	}
	for k, code := range raylib_keys {
		if rl.IsKeyDown(code) {
			in.Held[k] = true
		}
	}
	for k, b := range raylib_mouse {
		if rl.IsMouseButtonPressed(b) {
			in.Pressed[k] = true
			in.Held[k] = true
		}
		if rl.IsMouseButtonDown(b) {
			in.Held[k] = true
		}
	}
	if mouse_ok {
		d := rl.GetMouseDelta()
		in.Mouse_dx, in.Mouse_dy = float64(d.X), float64(d.Y)
	}
	p := rl.GetMousePosition()
	in.Mouse_x, in.Mouse_y = float64(p.X), float64(p.Y)
	in.Wheel = float64(rl.GetMouseWheelMove())
	return in
}
