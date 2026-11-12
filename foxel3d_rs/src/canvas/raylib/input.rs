use ::raylib::ffi as rl;
use ::raylib::ffi::KeyboardKey as RK;

use crate::canvas::{Input, Key};

#[rustfmt::skip]
fn keyboard_code(k: Key) -> Option<RK> {
    use Key::*;
    Some(match k {
        A => RK::KEY_A, B => RK::KEY_B, C => RK::KEY_C, D => RK::KEY_D, E => RK::KEY_E, F => RK::KEY_F,
        G => RK::KEY_G, H => RK::KEY_H, I => RK::KEY_I, J => RK::KEY_J, K => RK::KEY_K, L => RK::KEY_L,
        M => RK::KEY_M, N => RK::KEY_N, O => RK::KEY_O, P => RK::KEY_P, Q => RK::KEY_Q, R => RK::KEY_R,
        S => RK::KEY_S, T => RK::KEY_T, U => RK::KEY_U, V => RK::KEY_V, W => RK::KEY_W, X => RK::KEY_X,
        Y => RK::KEY_Y, Z => RK::KEY_Z,
        Num0 => RK::KEY_ZERO, Num1 => RK::KEY_ONE, Num2 => RK::KEY_TWO, Num3 => RK::KEY_THREE, Num4 => RK::KEY_FOUR,
        Num5 => RK::KEY_FIVE, Num6 => RK::KEY_SIX, Num7 => RK::KEY_SEVEN, Num8 => RK::KEY_EIGHT, Num9 => RK::KEY_NINE,
        Space => RK::KEY_SPACE, Enter => RK::KEY_ENTER, Escape => RK::KEY_ESCAPE, Tab => RK::KEY_TAB,
        Backspace => RK::KEY_BACKSPACE,
        LeftShift => RK::KEY_LEFT_SHIFT, RightShift => RK::KEY_RIGHT_SHIFT,
        LeftCtrl => RK::KEY_LEFT_CONTROL, RightCtrl => RK::KEY_RIGHT_CONTROL,
        LeftAlt => RK::KEY_LEFT_ALT, RightAlt => RK::KEY_RIGHT_ALT,
        Up => RK::KEY_UP, Down => RK::KEY_DOWN, Left => RK::KEY_LEFT, Right => RK::KEY_RIGHT,
        Slash => RK::KEY_SLASH, Minus => RK::KEY_MINUS, Equal => RK::KEY_EQUAL,
        Comma => RK::KEY_COMMA, Period => RK::KEY_PERIOD, Grave => RK::KEY_GRAVE,
        F1 => RK::KEY_F1, F2 => RK::KEY_F2, F3 => RK::KEY_F3, F4 => RK::KEY_F4, F5 => RK::KEY_F5, F6 => RK::KEY_F6,
        F7 => RK::KEY_F7, F8 => RK::KEY_F8, F9 => RK::KEY_F9, F10 => RK::KEY_F10, F11 => RK::KEY_F11, F12 => RK::KEY_F12,
        MouseLeft | MouseRight | MouseMiddle => return None,
    })
}

fn mouse_code(k: Key) -> Option<i32> {
    Some(match k {
        Key::MouseLeft => rl::MouseButton::MOUSE_BUTTON_LEFT as i32,
        Key::MouseRight => rl::MouseButton::MOUSE_BUTTON_RIGHT as i32,
        Key::MouseMiddle => rl::MouseButton::MOUSE_BUTTON_MIDDLE as i32,
        _ => return None,
    })
}

/// 上一帧鼠标的状态，用来丢掉突然很大的移动量
#[derive(Clone, Copy, Default, PartialEq)]
pub struct MouseState {
    /// 窗口有焦点，鼠标在窗口里或者被锁定
    tracking: bool,
    locked: bool,
}

/// 每帧画完后调用一次（EndDrawing里raylib刚处理完事件）
pub fn read_input(last: &mut MouseState) -> Input {
    let mut input = Input::default();
    unsafe {
        // 鼠标刚进入窗口、窗口刚获得焦点、刚锁定或解锁鼠标的那一帧，raylib是从旧位置算移动量的，会突然很大，丢掉这一帧的
        let mouse = MouseState {
            tracking: rl::IsWindowFocused() && (rl::IsCursorHidden() || rl::IsCursorOnScreen()),
            locked: rl::IsCursorHidden(),
        };
        let mouse_ok = mouse.tracking && mouse == *last;
        *last = mouse;
        if !rl::IsWindowFocused() {
            return input;
        }
        // raylib每帧处理一次事件：两帧之间按下又松开的键IsKeyDown看不到，但会留在GetKeyPressed的队列里
        loop {
            let code = rl::GetKeyPressed();
            if code == 0 {
                break;
            }
            if let Some(&k) = Key::ALL.iter().find(|&&k| keyboard_code(k).is_some_and(|c| c as i32 == code)) {
                input.pressed.insert(k);
                input.held.insert(k);
            }
        }
        for &k in Key::ALL {
            if let Some(c) = keyboard_code(k) {
                if rl::IsKeyDown(c as i32) {
                    input.held.insert(k);
                }
            } else if let Some(b) = mouse_code(k) {
                if rl::IsMouseButtonPressed(b) {
                    input.pressed.insert(k);
                    input.held.insert(k);
                }
                if rl::IsMouseButtonDown(b) {
                    input.held.insert(k);
                }
            }
        }
        loop {
            let ch = rl::GetCharPressed();
            if ch == 0 {
                break;
            }
            if let Some(c) = char::from_u32(ch as u32) {
                input.text.push(c);
            }
        }
        if mouse_ok {
            let d = rl::GetMouseDelta();
            input.mouse_dx = d.x as f64;
            input.mouse_dy = d.y as f64;
        }
        let p = rl::GetMousePosition();
        input.mouse_x = p.x as f64;
        input.mouse_y = p.y as f64;
        input.wheel = rl::GetMouseWheelMove() as f64;
    }
    input
}
