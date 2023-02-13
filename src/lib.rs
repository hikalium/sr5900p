#![feature(new_uninit)]
#![feature(slice_take)]
#![feature(exclusive_range_pattern)]
pub mod analyzer;
pub mod display;
pub mod print;
pub mod protocol;

use crate::print::mm_to_px;
use crate::protocol::PacketHeader;
use anyhow::anyhow;
use anyhow::Result;

#[derive(Copy, Clone, Debug, PartialEq, Eq)]
pub enum Tape {
    W4,
    W6,
    W9,
    W12,
    W18,
    W24,
    W36,
}
impl Tape {
    pub fn from_mm(mm: usize) -> Result<Self> {
        Ok(match mm {
            4 => Tape::W4,
            6 => Tape::W6,
            9 => Tape::W9,
            12 => Tape::W12,
            18 => Tape::W18,
            24 => Tape::W24,
            36 => Tape::W36,
            _ => return Err(anyhow!("Tape for {mm} mm is not defined")),
        })
    }
    fn width_px(&self) -> i32 {
        let w = match self {
            Tape::W4 => 2.85,  // verified
            Tape::W6 => 5.0,   // verified
            Tape::W9 => 7.0,   // verified
            Tape::W12 => 10.0, // verified
            Tape::W18 => 15.2, // verified
            Tape::W24 => 20.0, // verified
            Tape::W36 => 26.0, // verified
        };
        let w = mm_to_px(w);
        // tape width in px should be multiple of 8
        (w + 7) / 8 * 8
    }
}

#[derive(Copy, Clone, Debug)]
pub enum PrinterStatus {
    NoTape,
    SomeTape(Tape),
    CoverIsOpened,
    Printing,
    Unknown(PacketHeader, [u8; 20]),
}
