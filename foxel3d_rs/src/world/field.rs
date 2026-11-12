//! 场：给cube附加的自定义数据，比如箱子里装了什么、mod的状态。
//!
//! 场按名字分组（mod请用"mod名:场名"，避免和别人重名），key是CubeId，value是字符串，
//! 需要结构化的数据时自己编码成JSON之类的。
//! 场会跟着存档保存；cube被set_cube覆盖（包括挖掉）时，它里面的场会被删掉。

use std::collections::HashMap;

use super::{CubeId, World};

impl World {
    pub fn field_write(&mut self, field: &str, id: CubeId, value: String) {
        self.fields.entry(field.to_string()).or_default().insert(id, value);
    }

    pub fn field_read(&self, field: &str, id: CubeId) -> Option<&str> {
        self.fields.get(field)?.get(&id).map(String::as_str)
    }

    /// 不存在时什么都不做
    pub fn field_delete(&mut self, field: &str, id: CubeId) {
        if let Some(f) = self.fields.get_mut(field) {
            f.remove(&id);
        }
    }

    /// 某个场里的所有数据
    pub fn field_all(&self, field: &str) -> Vec<(CubeId, &str)> {
        match self.fields.get(field) {
            Some(f) => f.iter().map(|(k, v)| (*k, v.as_str())).collect(),
            None => Vec::new(),
        }
    }

    /// 所有场（存档用）
    pub fn fields(&self) -> &HashMap<String, HashMap<CubeId, String>> {
        &self.fields
    }

    pub fn set_fields(&mut self, fields: HashMap<String, HashMap<CubeId, String>>) {
        self.fields = fields;
    }
}
