// 这里是一些没法被归类的函数。

/// 返回src/下的 assets 路径。
pub fn get_assets_root() -> String {
    //
    format!("{}/src/assets", env!("CARGO_MANIFEST_DIR"))
}