use foxel::canvas::{
    canvas::{Canvas, Canvas1d, Canvas2d, CanvasColor, DrawIns},
    raylib,
};
use foxel::luabind;
use mlua::{Lua, Result};

/// ZZZZ
fn main() -> Result<()>{
    //println!(env!("CARGO_MANIFEST_DIR"));
    let lua = Lua::new();
    // Init the rust bind.
    let rust_table = lua.create_table()?;
    lua.globals().set("rust", rust_table)?;
    luabind::bind(&lua)?;
    /*lua.load(r#"
    print(_VERSION)
    print("hello from lua")
    for i=10, 100000000 do
      local j = i
      while j>10 do 
        if j%2==0 then j=j/2 else j=j*3+1 end
      end
      if i%1000000==0 then
        print(i)
      end
    end
    "#).exec()?;
    */
    lua.load(r#"
    print(rust.read_string("/init.lua"))
    print(load(rust.read_string("/init.lua"))())
    while true do end
    "#).exec()?;
    let mut r = raylib::RaylibCanvas::new((800, 600), true);
    loop {
        r = r.draw(vec![DrawIns::Square {
            side: Canvas1d(30),
            center: Canvas2d(50, 50),
            color: CanvasColor(100, 100, 100, 255),
            rotation: 0.0,
        }]);
        if r.should_close() {
            break;
        }
        println!("fff")
    }
    Ok(())
}
