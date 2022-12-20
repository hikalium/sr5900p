use crate::PrinterStatus;
use crate::TapeKind;
use anyhow::anyhow;
use anyhow::Context;
use anyhow::Result;
use std::boxed::Box;
use std::mem::size_of;
use std::mem::MaybeUninit;
use std::net::UdpSocket;
use std::slice;

/// # Safety
/// Implementing this trait is safe only when the target type can be converted
/// mutually between a byte sequence of the same size, which means that no ownership
/// nor memory references are involved.
pub unsafe trait Sliceable: Sized + Copy + Clone {
    fn copy_into_slice(&self) -> Box<[u8]> {
        let mut values = Box::<[u8]>::new_uninit_slice(size_of::<Self>());
        unsafe {
            values.copy_from_slice(slice::from_raw_parts(
                self as *const Self as *const MaybeUninit<u8>,
                size_of::<Self>(),
            ));
            values.assume_init()
        }
    }
    fn copy_from_slice(data: &[u8]) -> Result<Self> {
        if size_of::<Self>() > data.len() {
            Err(anyhow!("data is too short"))
        } else {
            Ok(unsafe { *(data.as_ptr() as *const Self) })
        }
    }
}
unsafe impl Sliceable for PacketHeader {}
unsafe impl Sliceable for StatusRequest {}
unsafe impl Sliceable for StartPrintRequest {}
unsafe impl Sliceable for StopPrintRequest {}

#[repr(packed)]
#[derive(Copy, Clone, Debug)]
pub struct PacketHeader {
    _signature: [u8; 4],  // "TPRT" for requests, "tprt" for responses
    _const00_be: [u8; 4], // 00 00 00 00
    _const01_be: [u8; 4], // 00 00 00 01
    _const20_be: [u8; 4], // 00 00 00 20
    _cmd_be: [u8; 4],
    _data_size_be: [u8; 4],
    _ip_addr_be: [u8; 4],
    _token_be: [u8; 4],
}
impl PacketHeader {
    pub fn new_request(cmd: u32, data_size: u32) -> Self {
        Self {
            _signature: *b"TPRT",
            _const00_be: 0x00u32.to_be_bytes(),
            _const01_be: 0x01u32.to_be_bytes(),
            _const20_be: 0x20u32.to_be_bytes(),
            _cmd_be: cmd.to_be_bytes(),
            _data_size_be: data_size.to_be_bytes(),
            _ip_addr_be: 0x00u32.to_be_bytes(),
            _token_be: 0x00u32.to_be_bytes(),
        }
    }
}

#[repr(packed)]
#[derive(Copy, Clone)]
pub struct StatusRequest {
    _header: PacketHeader,
}
impl StatusRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(1, 0),
        }
    }
    pub fn send(socket: &UdpSocket, device_ip: &str) -> Result<PrinterStatus> {
        let req = Self::new();
        socket
            .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
            .context("failed to send")?;
        let mut buf = [0; 128];
        let (len, _) = socket.recv_from(&mut buf)?;
        let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
        let data = &buf[size_of::<PacketHeader>()..len];
        println!("{:?}", data);
        // idle
        // [20, 0,  0, 4, 0, 0, 0, 0, 64, 0, 0,  0, 0, 0,  0, 0,  0, 0, 0, 0]
        // printing
        // [20, 2,  0, 4, 0, 0, 0, 0, 64, 0, 0,  0, 0, 0,  0, 0,  0, 0, 0, 0]
        // printing completed
        // [20, 0,  0, 4, 0, 0, 0, 0, 64, 0, 0,  0, 0, 1,  0, 0,  0, 0, 0, 0]
        // Tape exhausted
        // [20, 0, 66, 4, 0, 0, 0, 0, 64, 0, 0, 64, 0, 0, 66, 0, 64, 0, 0, 0]
        // ???
        // [20, 0,  0, 4, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0,  66, 0, 64, 0, 0, 0]
        let data: [u8; 20] = data.try_into().context(anyhow!(
            "invalid data len. expected 20 but got {}",
            data.len()
        ))?;
        Ok(match (data[0x01], data[0x0d]) {
            (2, 0) => PrinterStatus::Printing,
            (0, 0 | 1 | 2) => match data[0x02] {
                0x06 => PrinterStatus::NoTape,
                0x21 => PrinterStatus::CoverIsOpened,
                0x00 => PrinterStatus::SomeTape(match data[0x03] {
                    0x01 => TapeKind::W6,
                    0x02 => TapeKind::W9,
                    0x03 => TapeKind::W12,
                    0x04 => TapeKind::W18,
                    0x05 => TapeKind::W24,
                    0x06 => TapeKind::W36,
                    ti => TapeKind::UnknownTapeIndex(ti),
                }),
                _ => PrinterStatus::Unknown(res_header, data),
            },
            (v01, v0d) => {
                eprintln!("v01: {v01}, v0d: {v0d}");
                PrinterStatus::Unknown(res_header, data)
            }
        })
    }
}

#[repr(packed)]
#[derive(Copy, Clone)]
pub struct StartPrintRequest {
    _header: PacketHeader,
}
impl StartPrintRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(2, 0),
        }
    }
    pub fn send(socket: &UdpSocket, device_ip: &str) -> Result<()> {
        let req = Self::new();
        socket
            .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
            .context("failed to send")?;
        let mut buf = [0; 128];
        let (len, _) = socket.recv_from(&mut buf)?;
        let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
        let data = &buf[size_of::<PacketHeader>()..len];
        if data == [2, 0, 0] {
            Ok(())
        } else {
            Err(anyhow!(
                "Failed to start printing. res_header: {:?}, data: {:?}",
                res_header,
                data
            ))
        }
    }
}

#[repr(packed)]
#[derive(Copy, Clone)]
pub struct StopPrintRequest {
    _header: PacketHeader,
}
impl StopPrintRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(3, 0),
        }
    }
    pub fn send(socket: &UdpSocket, device_ip: &str) -> Result<()> {
        let req = Self::new();
        socket
            .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
            .context("failed to send")?;
        let mut buf = [0; 128];
        let (len, _) = socket.recv_from(&mut buf)?;
        let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
        let data = &buf[size_of::<PacketHeader>()..len];
        if data == [3, 0, 0] {
            Ok(())
        } else {
            Err(anyhow!(
                "Failed to stop printing. res_header: {:?}, data: {:?}",
                res_header,
                data
            ))
        }
    }
}

pub fn notify_data_stream(socket: &UdpSocket, device_ip: &str) -> Result<()> {
    let mut buf = [0; 128];

    let req = PacketHeader::new_request(0x0101, 0);
    socket
        .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
        .context("failed to send")?;
    let (len, _) = socket.recv_from(&mut buf)?;
    let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
    let data = &buf[size_of::<PacketHeader>()..len];
    if data.len() != 0 {
        return Err(anyhow!(
            "Invalid response for cmd 0101: {:?}, data: {:?}",
            res_header,
            data
        ));
    }

    let req = PacketHeader::new_request(0x0100, 0);
    socket
        .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
        .context("failed to send")?;
    let (len, _) = socket.recv_from(&mut buf)?;
    let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
    let data = &buf[size_of::<PacketHeader>()..len];
    if data == [0x00] {
        println!("Warning: response for cmd 0x0100 was 0x00 (normally 0x10)");
    } else if data != [0x10] {
        return Err(anyhow!(
            "Invalid response for cmd 0100: {:?}, data: {:?}",
            res_header,
            data
        ));
    }
    Ok(())
}
