#![feature(new_uninit)]

use anyhow::anyhow;
use anyhow::Context;
use anyhow::Result;
use argh::FromArgs;
use std::boxed::Box;
use std::fs;
use std::io::prelude::Write;
use std::mem::size_of;
use std::mem::MaybeUninit;
use std::net::TcpStream;
use std::net::UdpSocket;
use std::slice;
use std::thread;
use std::time;

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
struct PacketHeader {
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
    fn new_request(cmd: u32, data_size: u32) -> Self {
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

#[derive(Copy, Clone, Debug)]
enum TapeKind {
    W6,
    W9,
    W12,
    W18,
    W24,
    W36,
    UnknownTapeIndex(u8),
}

#[derive(Copy, Clone, Debug)]
enum PrinterStatus {
    NoTape,
    SomeTape(TapeKind),
    CoverIsOpened,
    Printing,
    Unknown(PacketHeader, [u8; 20]),
}

#[repr(packed)]
#[derive(Copy, Clone)]
struct StatusRequest {
    _header: PacketHeader,
}
impl StatusRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(1, 0),
        }
    }
    fn send(socket: &UdpSocket, device_ip: &str) -> Result<PrinterStatus> {
        let req = Self::new();
        socket
            .send_to(&req.copy_into_slice(), device_ip.to_string() + ":9100")
            .context("failed to send")?;
        let mut buf = [0; 128];
        let (len, _) = socket.recv_from(&mut buf)?;
        let res_header = PacketHeader::copy_from_slice(&buf[0..len])?;
        let data = &buf[size_of::<PacketHeader>()..len];
        println!("{:?}", data);
        // [20, 0, 0, 4, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0] // idle
        // [20, 2, 0, 4, 0, 0, 0, 0, 64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0] // printing
        // [20, 0, 0, 4, 0, 0, 0, 0, 64, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0] // print is done
        let data: [u8; 20] = data.try_into().context(anyhow!(
            "invalid data len. expected 20 but got {}",
            data.len()
        ))?;
        Ok(match (data[0x01], data[0x0d]) {
            (2, 0) => PrinterStatus::Printing,
            (0, 1) => match data[0x02] {
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
            _ => PrinterStatus::Unknown(res_header, data),
        })
    }
}

#[repr(packed)]
#[derive(Copy, Clone)]
struct StartPrintRequest {
    _header: PacketHeader,
}
impl StartPrintRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(2, 0),
        }
    }
    fn send(socket: &UdpSocket, device_ip: &str) -> Result<()> {
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
struct StopPrintRequest {
    _header: PacketHeader,
}
impl StopPrintRequest {
    fn new() -> Self {
        Self {
            _header: PacketHeader::new_request(3, 0),
        }
    }
    fn send(socket: &UdpSocket, device_ip: &str) -> Result<()> {
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

fn notify_data_stream(socket: &UdpSocket, device_ip: &str) -> Result<()> {
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
    if data != [0x10] {
        return Err(anyhow!(
            "Invalid response for cmd 0100: {:?}, data: {:?}",
            res_header,
            data
        ));
    }
    Ok(())
}

fn do_analyze(_dump_file: &str) -> Result<()> {
    let label_data = fs::read("sample_tcp_data.bin")?;
    println!("Size: {}", label_data.len());
    let mut i = 0;
    while i < label_data.len() {
        match label_data[i] {
            0x1b => match label_data[i + 1] {
                0x7b => {
                    let len = label_data[i + 2] as usize;
                    println!(
                        "cmd 0x1b 0x7b, len = {}, {:?}",
                        len,
                        &label_data[i + 3..i + 3 + len]
                    );
                    i += 3 + len;
                }
                0x2e => {
                    if label_data[i + 2..i + 6] != [0, 0, 0, 1] {
                        return Err(anyhow!(
                            "Unexpected label data: {:?}...",
                            &label_data[i..i + 16]
                        ));
                    }
                    let bits = label_data[i + 6] as usize + label_data[i + 7] as usize * 256;
                    let bytes = (bits + 7) / 8;
                    print!("cmd 0x1b 0x2e, bits = {bits}, bytes = {bytes}: ",);
                    let img_data = &label_data[i + 8..i + 8 + bytes];
                    for byte in img_data {
                        print!("{byte:08b}");
                    }
                    print!("\n");
                    i += 8 + bytes;
                }
                _ => {
                    return Err(anyhow!(
                        "Unexpected label data: {:?}...",
                        &label_data[i..i + 16]
                    ));
                }
            },
            0x0c => {
                println!("cmd 0x0c (data end marker?)",);
                i += 1;
            }
            _ => {
                return Err(anyhow!("Unexpected label data: {:?}...", &label_data[i..]));
            }
        }
    }
    Ok(())
}

fn do_print(device_ip: &str) -> Result<()> {
    let socket = UdpSocket::bind("0.0.0.0:0").context("failed to bind")?;
    let info = StatusRequest::send(&socket, device_ip)?;
    println!("{:?}", info);
    if let PrinterStatus::SomeTape(t) = info {
        println!("Tape is {:?}, start printing...", t);
    } else {
        println!("Unexpected state. Aborting...");
        std::process::exit(1);
    }
    StartPrintRequest::send(&socket, device_ip)?;
    thread::sleep(time::Duration::from_millis(500));
    let mut stream = TcpStream::connect(device_ip.to_string() + ":9100")?;
    thread::sleep(time::Duration::from_millis(500));
    notify_data_stream(&socket, device_ip)?;
    thread::sleep(time::Duration::from_millis(500));
    let label_data = fs::read("sample_tcp_data.bin")?;
    stream.write(&label_data)?;

    println!("Print data is sent. Waiting...");
    loop {
        thread::sleep(time::Duration::from_millis(500));
        let info = StatusRequest::send(&socket, device_ip)?;
        println!("{:?}", info);
        if let PrinterStatus::Printing = info {
            continue;
        }
        break;
    }

    StopPrintRequest::send(&socket, device_ip)?;

    Ok(())
}

#[derive(Debug, FromArgs)]
/// Reach new heights.
struct Args {
    /// for debug: analyze the print data sent via TCP (specify the raw dump of the TCP stream)
    #[argh(option)]
    analyze_tcp_data: Option<String>,

    /// an IPv4 address for the printer
    #[argh(positional)]
    device_ip: Option<String>,
}

fn main() -> Result<()> {
    let args: Args = argh::from_env();
    println!("{:?}", args);

    if let Some(dump_file) = args.analyze_tcp_data {
        do_analyze(&dump_file)
    } else if let Some(device_ip) = args.device_ip {
        do_print(&device_ip)
    } else {
        Err(anyhow!("Unknown command"))
    }
}
