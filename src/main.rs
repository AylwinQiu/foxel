use foxel::canvas::{
    canvas::{Canvas, Canvas1d, Canvas2d, CanvasColor, DrawIns},
    raylib,
};
use rustpython_vm::Interpreter;

/// ZZZZ
fn main() {
    Interpreter::without_stdlib(Default::default()).enter(|vm| {
        // Your vm can run in here.
        vm.run_simple_string(
            r#"
print("hello from python!")

a = 10
b = 20
print(a + b)

for i in range(5000000):
    print(i)
"#,
        )
        .unwrap();
    });

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
}
