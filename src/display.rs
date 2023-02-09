use embedded_graphics::pixelcolor::BinaryColor;
use embedded_graphics::prelude::DrawTarget;
use embedded_graphics::prelude::OriginDimensions;
use embedded_graphics::prelude::Size;
use embedded_graphics::Pixel;

pub struct TapeDisplay {
    pub framebuffer: Vec<Vec<bool>>,
    pub width: usize,
    pub height: usize,
}
impl TapeDisplay {
    pub fn new(width: usize, height: usize) -> Self {
        let mut row = Vec::new();
        row.resize(width, false);
        let mut framebuffer = Vec::new();
        framebuffer.resize(height, row);
        Self {
            framebuffer,
            width,
            height,
        }
    }
    pub fn scaled(&self, r: usize) -> Self {
        let mut new = Self::new(self.width * r, self.height * r);
        for y in 0..new.height {
            for x in 0..new.width {
                new.set_pixel(x, y, self.get_pixel(x / r, y / r));
            }
        }
        new
    }
    pub fn rotated(&self) -> Self {
        // 90 deg left (counter-clockwise)
        let mut new = Self::new(self.height, self.width);
        for y in 0..new.height {
            for x in 0..new.width {
                new.set_pixel(x, y, self.get_pixel(new.height - 1 - y, x));
            }
        }
        new
    }
    pub fn overlay_or(&mut self, td: &Self, px: usize, py: usize) {
        for y in py..self.height {
            for x in px..self.width {
                self.framebuffer[y][x] |= td
                    .framebuffer
                    .get(y - py)
                    .and_then(|r| r.get(x - px))
                    .unwrap_or(&false);
            }
        }
    }
    pub fn get_pixel(&self, x: usize, y: usize) -> bool {
        *self
            .framebuffer
            .get(y)
            .and_then(|r| r.get(x))
            .unwrap_or(&false)
    }
    pub fn set_pixel(&mut self, x: usize, y: usize, value: bool) {
        if let Some(v) = self.framebuffer.get_mut(y).and_then(|r| r.get_mut(x)) {
            *v = value
        }
    }
}
impl DrawTarget for TapeDisplay {
    type Color = BinaryColor;
    type Error = core::convert::Infallible;

    fn draw_iter<I>(&mut self, pixels: I) -> Result<(), Self::Error>
    where
        I: IntoIterator<Item = Pixel<Self::Color>>,
    {
        let w = self.width as i32;
        let h = self.height as i32;
        for Pixel(coord, color) in pixels.into_iter() {
            let (x, y) = coord.into();
            if (0..w).contains(&x) && (0..h).contains(&y) {
                self.framebuffer[y as usize][x as usize] = color.is_on();
            }
        }
        Ok(())
    }
}
impl OriginDimensions for TapeDisplay {
    fn size(&self) -> Size {
        Size::new(self.width as u32, self.height as u32)
    }
}

#[test]
fn transforms() {
    // 2x2
    let mut td = TapeDisplay::new(2, 2);
    // 0 0
    // 0 0
    assert_eq!(td.get_pixel(0, 0), false);
    assert_eq!(td.get_pixel(0, 1), false);
    assert_eq!(td.get_pixel(1, 0), false);
    assert_eq!(td.get_pixel(1, 1), false);
    td.set_pixel(0, 0, true);
    td.set_pixel(1, 1, true);
    // 1 0
    // 0 1
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 1), false);
    assert_eq!(td.get_pixel(1, 0), false);
    assert_eq!(td.get_pixel(1, 1), true);
    let td = td.scaled(2);
    // 1 1 0 0
    // 1 1 0 0
    // 0 0 1 1
    // 0 0 1 1
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 2), false);
    assert_eq!(td.get_pixel(2, 0), false);
    assert_eq!(td.get_pixel(2, 2), true);
    let td = td.rotated();
    // 0 0 1 1
    // 0 0 1 1
    // 1 1 0 0
    // 1 1 0 0
    assert_eq!(td.get_pixel(0, 0), false);
    assert_eq!(td.get_pixel(0, 2), true);
    assert_eq!(td.get_pixel(2, 0), true);
    assert_eq!(td.get_pixel(2, 2), false);
    let td = td.rotated();
    // 1 1 0 0
    // 1 1 0 0
    // 0 0 1 1
    // 0 0 1 1
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 2), false);
    assert_eq!(td.get_pixel(2, 0), false);
    assert_eq!(td.get_pixel(2, 2), true);

    // 3x2
    let mut td = TapeDisplay::new(3, 2);
    // 0 0 0
    // 0 0 0
    assert_eq!(td.get_pixel(0, 0), false);
    assert_eq!(td.get_pixel(0, 1), false);
    assert_eq!(td.get_pixel(1, 0), false);
    assert_eq!(td.get_pixel(1, 1), false);
    assert_eq!(td.get_pixel(2, 0), false);
    assert_eq!(td.get_pixel(2, 1), false);
    td.set_pixel(0, 0, true);
    td.set_pixel(1, 1, true);
    td.set_pixel(2, 0, true);
    // 1 0 1
    // 0 1 0
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 1), false);
    assert_eq!(td.get_pixel(1, 0), false);
    assert_eq!(td.get_pixel(1, 1), true);
    assert_eq!(td.get_pixel(2, 0), true);
    assert_eq!(td.get_pixel(2, 1), false);
    let td = td.scaled(2);
    // 1 1 0 0 1 1
    // 1 1 0 0 1 1
    // 0 0 1 1 0 0
    // 0 0 1 1 0 0
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 2), false);
    assert_eq!(td.get_pixel(2, 0), false);
    assert_eq!(td.get_pixel(2, 2), true);
    assert_eq!(td.get_pixel(4, 0), true);
    assert_eq!(td.get_pixel(4, 2), false);
    let td = td.rotated();
    // 1 1 0 0
    // 1 1 0 0
    // 0 0 1 1
    // 0 0 1 1
    // 1 1 0 0
    // 1 1 0 0
    println!("{:?}", td.framebuffer);
    assert_eq!(td.get_pixel(0, 0), true);
    assert_eq!(td.get_pixel(0, 2), false);
    assert_eq!(td.get_pixel(2, 0), false);
    assert_eq!(td.get_pixel(2, 2), true);
    assert_eq!(td.get_pixel(0, 4), true);
    assert_eq!(td.get_pixel(2, 4), false);
}
