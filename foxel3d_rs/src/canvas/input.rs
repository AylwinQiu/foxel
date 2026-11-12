use std::collections::HashSet;

macro_rules! keys {
    ($($key:ident => $name:literal),* $(,)?) => {
        /// 键盘和鼠标的键。在Lua里用name()这个名字，比如"w"、"space"、"mouse_left"
        #[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
        pub enum Key { $($key),* }

        impl Key {
            pub const ALL: &[Key] = &[$(Key::$key),*];

            pub fn name(self) -> &'static str {
                match self { $(Key::$key => $name),* }
            }

            pub fn from_name(name: &str) -> Option<Key> {
                match name { $($name => Some(Key::$key),)* _ => None }
            }
        }
    };
}

keys! {
    A => "a", B => "b", C => "c", D => "d", E => "e", F => "f", G => "g", H => "h", I => "i",
    J => "j", K => "k", L => "l", M => "m", N => "n", O => "o", P => "p", Q => "q", R => "r",
    S => "s", T => "t", U => "u", V => "v", W => "w", X => "x", Y => "y", Z => "z",
    Num0 => "0", Num1 => "1", Num2 => "2", Num3 => "3", Num4 => "4",
    Num5 => "5", Num6 => "6", Num7 => "7", Num8 => "8", Num9 => "9",
    Space => "space", Enter => "enter", Escape => "escape", Tab => "tab", Backspace => "backspace",
    LeftShift => "left_shift", RightShift => "right_shift",
    LeftCtrl => "left_ctrl", RightCtrl => "right_ctrl",
    LeftAlt => "left_alt", RightAlt => "right_alt",
    Up => "up", Down => "down", Left => "left", Right => "right",
    Slash => "slash", Minus => "minus", Equal => "equal", Comma => "comma", Period => "period", Grave => "grave",
    F1 => "f1", F2 => "f2", F3 => "f3", F4 => "f4", F5 => "f5", F6 => "f6",
    F7 => "f7", F8 => "f8", F9 => "f9", F10 => "f10", F11 => "f11", F12 => "f12",
    MouseLeft => "mouse_left", MouseRight => "mouse_right", MouseMiddle => "mouse_middle",
}

/// 一帧的输入
#[derive(Clone, Debug, Default)]
pub struct Input {
    /// 这一帧按着的键和鼠标键，包括这一帧内按下又松开的
    pub held: HashSet<Key>,
    /// 这一帧刚按下的键和鼠标键
    pub pressed: HashSet<Key>,
    /// 这一帧鼠标移动了多少像素，x向右、y向下为正。鼠标刚进入窗口、刚锁定的那一帧是0
    pub mouse_dx: f64,
    pub mouse_dy: f64,
    /// 鼠标在窗口里的位置（像素）
    pub mouse_x: f64,
    pub mouse_y: f64,
    /// 滚轮，向上滚为正
    pub wheel: f64,
    /// 这一帧输入的文字（已经按键盘布局转换好了），给聊天框、命令行用
    pub text: String,
}

impl Input {
    pub fn is_held(&self, k: Key) -> bool {
        self.held.contains(&k)
    }

    pub fn is_pressed(&self, k: Key) -> bool {
        self.pressed.contains(&k)
    }
}
