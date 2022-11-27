use anyhow::Context;
use anyhow::Result;
use argh::FromArgs;
use std::net::UdpSocket;

#[derive(Debug, FromArgs)]
/// Reach new heights.
struct Args {
    /// whether or not to jump
    #[argh(switch, short = 'j')]
    jump: bool,

    /// how high to go
    #[argh(option)]
    height: Option<usize>,

    /// an optional nickname for the pilot
    #[argh(positional)]
    device_ip: String,
}

fn main() -> Result<()> {
    let args: Args = argh::from_env();
    println!("{:?}", args);

    let socket = UdpSocket::bind("0.0.0.0:0").context("failed to bind")?;
    let mut buf = [0; 128];
    let req_status = [
        0x54, 0x50, 0x52, 0x54, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
        0x20, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x0a, 0x0a, 0x5a, 0xb8, 0x60,
        0xe9, 0x3c,
    ];
    socket
        .send_to(&req_status, args.device_ip + ":9100")
        .context("failed to send")?;
    println!("sent!");
    let (amt, src) = socket.recv_from(&mut buf)?;
    println!("recv!");
    println!("{} {} {:?}", src, amt, &buf[0..amt]);

    Ok(())
}
