#![feature(new_uninit)]
#![feature(slice_take)]
#![feature(exclusive_range_pattern)]
pub mod analyzer;
pub mod display;
pub mod print;
pub mod protocol;

use crate::protocol::PacketHeader;

#[derive(Copy, Clone, Debug)]
pub enum TapeKind {
    W6,
    W9,
    W12,
    W18,
    W24,
    W36,
    UnknownTapeIndex(u8),
}

#[derive(Copy, Clone, Debug)]
pub enum PrinterStatus {
    NoTape,
    SomeTape(TapeKind),
    CoverIsOpened,
    Printing,
    Unknown(PacketHeader, [u8; 20]),
}
