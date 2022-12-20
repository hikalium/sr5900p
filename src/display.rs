use embedded_graphics::pixelcolor::BinaryColor;
use embedded_graphics::prelude::DrawTarget;
use embedded_graphics::prelude::OriginDimensions;
use embedded_graphics::prelude::Size;
use embedded_graphics::Pixel;

pub struct TapeDisplay {
    pub framebuffer: Vec<Vec<bool>>,
    width: usize,
    height: usize,
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
