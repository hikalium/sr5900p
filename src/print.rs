use crate::analyzer::analyze_tcp_data;
use crate::display::TapeDisplay;
use crate::protocol::notify_data_stream;
use crate::protocol::StartPrintRequest;
use crate::protocol::StatusRequest;
use crate::protocol::StopPrintRequest;
use crate::PrinterStatus;
use crate::TapeKind;
use anyhow::anyhow;
use anyhow::Context;
use anyhow::Result;
use argh::FromArgs;
use barcoders::sym::code39::Code39;
use embedded_graphics::geometry::Dimensions;
use embedded_graphics::geometry::Point;
use embedded_graphics::mono_font::ascii::FONT_10X20;
use embedded_graphics::mono_font::MonoTextStyle;
use embedded_graphics::pixelcolor::BinaryColor;
use embedded_graphics::prelude::Size;
use embedded_graphics::primitives::PrimitiveStyle;
use embedded_graphics::primitives::Rectangle;
use embedded_graphics::primitives::StyledDrawable;
use embedded_graphics::text::Alignment;
use embedded_graphics::text::Text;
use embedded_graphics::Drawable;
use image::Luma;
use qrcode::QrCode;
use regex::Regex;
use std::fs;
use std::fs::File;
use std::io::prelude::Write;
use std::io::BufWriter;
use std::net::TcpStream;
use std::net::UdpSocket;
use std::num::Wrapping;
use std::path::Path;
use std::thread;
use std::time;

#[derive(FromArgs, PartialEq, Debug)]
/// Print something
#[argh(subcommand, name = "print")]
pub struct PrintArgs {
    /// generate a label for a mac addr
    #[argh(option)]
    mac_addr: Option<String>,
    /// generate a label for a QR code with text
    #[argh(option)]
    qr_text: Option<String>,
    /// do not print (just generate and analyze)
    #[argh(switch)]
    dry_run: bool,
    /// the raw dump of the TCP stream while printing
    #[argh(option)]
    tcp_data: Option<String>,
    /// an IPv4 address for the printer
    #[argh(option)]
    printer: String,
}
fn print_tcp_data(device_ip: &str, data: &[u8]) -> Result<()> {
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
    stream.write(&data)?;

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
fn gen_tcp_data(td: &TapeDisplay) -> Result<Vec<u8>> {
    let mut tcp_data: Vec<u8> = Vec::new();
    tcp_data.append(&mut vec![27, 123, 3, 64, 64, 125]);
    tcp_data.append(&mut vec![27, 123, 7, 123, 0, 0, 83, 84, 34, 125]);
    tcp_data.append(&mut vec![27, 123, 7, 67, 2, 2, 1, 1, 73, 125]); // half-cut?
    tcp_data.append(&mut vec![27, 123, 4, 68, 5, 73, 125]);
    tcp_data.append(&mut vec![27, 123, 3, 71, 71, 125]);

    let mut tape_len_bytes = (td.width as u32).to_le_bytes().to_vec();
    let mut cmd_bytes = vec![76];
    cmd_bytes.append(&mut tape_len_bytes);
    let csum = cmd_bytes
        .iter()
        .map(|v| Wrapping(*v))
        .sum::<Wrapping<u8>>()
        .0;
    cmd_bytes.push(csum);
    cmd_bytes.push(0x7d);
    tcp_data.append(&mut vec![0x1b, 0x7b, cmd_bytes.len() as u8]);
    tcp_data.append(&mut cmd_bytes);

    tcp_data.append(&mut vec![27, 123, 5, 84, 42, 0, 126, 125]);
    tcp_data.append(&mut vec![27, 123, 4, 72, 5, 77, 125]);
    tcp_data.append(&mut vec![27, 123, 4, 115, 0, 115, 125]);

    let row_bytes = (td.height + 7) / 8;
    for y in 0..td.width {
        tcp_data.append(&mut vec![0x1b, 0x2e, 0, 0, 0, 1]);
        tcp_data.append(&mut (td.height as u16).to_le_bytes().to_vec());
        for xb in 0..row_bytes {
            let mut chunk = 0x00;
            for dx in 0..8 {
                let x = xb * 8 + (7 - dx);

                if td.get_pixel(td.width - 1 - y, x) {
                    chunk = chunk | (1 << dx)
                }
            }
            tcp_data.push(chunk);
        }
    }
    tcp_data.push(0x0c); // data end
    tcp_data.append(&mut vec![27, 123, 3, 64, 64, 125]);
    Ok(tcp_data)
}

fn print_mac_addr(mac_addr: &str, device_ip: &str, dry_run: bool) -> Result<()> {
    let text = mac_addr.to_uppercase().replace(":", "");
    println!("{:?}", text);
    let re = Regex::new(r"^[0-9A-Z]{12}$").unwrap();
    if !re.is_match(&text) {
        return Err(anyhow!("Invalid MAC Address: {mac_addr}"));
    }
    let socket = UdpSocket::bind("0.0.0.0:0").context("failed to bind")?;
    let info = StatusRequest::send(&socket, device_ip)?;
    let tape_width_px = if let PrinterStatus::SomeTape(t) = info {
        println!("Tape is {:?}", t);
        match t {
            // -4mm will be the printable width...
            TapeKind::W9 => 5 * 360 * 10 / 254,
            TapeKind::W12 => 10 * 360 * 10 / 254,
            TapeKind::W18 => 14 * 360 * 10 / 254, // verified
            TapeKind::W24 => 20 * 360 * 10 / 254,
            TapeKind::W36 => 26 * 360 * 10 / 254, // verified
            _ => return Err(anyhow!("Failed to calc tape width. status: {:?}", info)),
        }
    } else {
        return Err(anyhow!(
            "Failed to determine tape width. status: {:?}",
            info
        ));
    };
    let tape_width_px = (tape_width_px + 7) / 8 * 8;

    let qr_td = {
        let mut td = TapeDisplay::new(tape_width_px, tape_width_px);
        let tape_width_px = tape_width_px as u32;
        let code = QrCode::new(&text).unwrap();
        let image = code
            .render::<Luma<u8>>()
            .max_dimensions(tape_width_px, tape_width_px)
            .build();
        let ofs_x = (tape_width_px - image.width()) / 2;
        let ofs_y = (tape_width_px - image.height()) / 2;
        for (x, y, p) in image.enumerate_pixels() {
            Rectangle::new(
                Point::new((x + ofs_x) as i32, (y + ofs_y) as i32),
                Size::new_equal(1),
            )
            .draw_styled(
                &PrimitiveStyle::with_fill(BinaryColor::from(p.0[0] == 0)),
                &mut td,
            )?;
        }
        image.save("qrcode.png").unwrap();
        td
    };

    let barcode = Code39::new(&text).context("Failed to generate a barcode")?;
    let encoded: Vec<u8> = barcode.encode();
    println!("{:?}", encoded);

    let mac_td = {
        let character_style = MonoTextStyle::new(&FONT_10X20, BinaryColor::On);
        let text_len = text.len();
        let mut td = TapeDisplay::new(10 * text_len, 20);
        Text::with_alignment(
            &text,
            td.bounding_box().center() + Point::new(0, 10),
            character_style,
            Alignment::Center,
        )
        .draw(&mut td)?;
        let td = td.rotated();
        let r = qr_td.height / td.height;
        td.scaled(r)
    };

    // Merge the components
    let mut td = TapeDisplay::new(qr_td.width + mac_td.width, tape_width_px);
    td.overlay_or(&mac_td, 0, 0);
    td.overlay_or(&qr_td, mac_td.width, 0);

    // Generate preview image
    let path = Path::new(r"preview.png");
    let file = File::create(path).unwrap();
    let ref mut w = BufWriter::new(file);
    let mut encoder = png::Encoder::new(w, td.width as u32, td.height as u32); // Width is 2 pixels and height is 1.
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    encoder.set_source_gamma(png::ScaledFloat::from_scaled(45455)); // 1.0 / 2.2, scaled by 100000
    encoder.set_source_gamma(png::ScaledFloat::new(1.0 / 2.2)); // 1.0 / 2.2, unscaled, but rounded
    let source_chromaticities = png::SourceChromaticities::new(
        // Using unscaled instantiation here
        (0.31270, 0.32900),
        (0.64000, 0.33000),
        (0.30000, 0.60000),
        (0.15000, 0.06000),
    );
    encoder.set_source_chromaticities(source_chromaticities);
    let mut writer = encoder.write_header().unwrap();
    let data: Vec<u8> = td
        .framebuffer
        .iter()
        .flat_map(|row| row.iter())
        .flat_map(|c| {
            // data will be [RGBARGBA...]
            if *c {
                [0, 0, 0, 255]
            } else {
                [255, 255, 255, 255]
            }
        })
        .collect();
    writer.write_image_data(&data).unwrap();

    let tcp_data = gen_tcp_data(&td)?;

    if !dry_run {
        print_tcp_data(device_ip, &tcp_data)
    } else {
        analyze_tcp_data(&tcp_data)?;
        Ok(())
    }
}

fn print_qr_text(text: &str, device_ip: &str, dry_run: bool) -> Result<()> {
    println!("{:?}", text);
    let socket = UdpSocket::bind("0.0.0.0:0").context("failed to bind")?;
    let info = StatusRequest::send(&socket, device_ip)?;
    let tape_width_px = if let PrinterStatus::SomeTape(t) = info {
        println!("Tape is {:?}", t);
        match t {
            // -4mm will be the printable width...
            TapeKind::W9 => 5 * 360 * 10 / 254,
            TapeKind::W12 => 10 * 360 * 10 / 254,
            TapeKind::W18 => 14 * 360 * 10 / 254, // verified
            TapeKind::W24 => 20 * 360 * 10 / 254,
            TapeKind::W36 => 26 * 360 * 10 / 254, // verified
            _ => return Err(anyhow!("Failed to calc tape width. status: {:?}", info)),
        }
    } else {
        return Err(anyhow!(
            "Failed to determine tape width. status: {:?}",
            info
        ));
    };
    let tape_width_px = (tape_width_px + 7) / 8 * 8;

    let qr_td = {
        let mut td = TapeDisplay::new(tape_width_px, tape_width_px);
        let tape_width_px = tape_width_px as u32;
        let code = QrCode::new(&text).unwrap();
        let image = code
            .render::<Luma<u8>>()
            .max_dimensions(tape_width_px, tape_width_px)
            .build();
        let ofs_x = (tape_width_px - image.width()) / 2;
        let ofs_y = (tape_width_px - image.height()) / 2;
        for (x, y, p) in image.enumerate_pixels() {
            Rectangle::new(
                Point::new((x + ofs_x) as i32, (y + ofs_y) as i32),
                Size::new_equal(1),
            )
            .draw_styled(
                &PrimitiveStyle::with_fill(BinaryColor::from(p.0[0] == 0)),
                &mut td,
            )?;
        }
        image.save("qrcode.png").unwrap();
        td
    };

    let text_td = {
        let character_style = MonoTextStyle::new(&FONT_10X20, BinaryColor::On);
        let text_len = text.len();
        let mut td = TapeDisplay::new(10 * text_len, 20);
        Text::with_alignment(
            &text,
            td.bounding_box().center() + Point::new(0, 10),
            character_style,
            Alignment::Center,
        )
        .draw(&mut td)?;
        let r = tape_width_px / td.height;
        td.scaled(r)
    };

    // Merge the components
    let mut td = TapeDisplay::new(qr_td.width + text_td.width, tape_width_px);
    td.overlay_or(&qr_td, 0, 0);
    td.overlay_or(&text_td, qr_td.width, (tape_width_px - text_td.height) / 2);

    // Generate preview image
    let path = Path::new(r"preview.png");
    let file = File::create(path).unwrap();
    let ref mut w = BufWriter::new(file);
    let mut encoder = png::Encoder::new(w, td.width as u32, td.height as u32); // Width is 2 pixels and height is 1.
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    encoder.set_source_gamma(png::ScaledFloat::from_scaled(45455)); // 1.0 / 2.2, scaled by 100000
    encoder.set_source_gamma(png::ScaledFloat::new(1.0 / 2.2)); // 1.0 / 2.2, unscaled, but rounded
    let source_chromaticities = png::SourceChromaticities::new(
        // Using unscaled instantiation here
        (0.31270, 0.32900),
        (0.64000, 0.33000),
        (0.30000, 0.60000),
        (0.15000, 0.06000),
    );
    encoder.set_source_chromaticities(source_chromaticities);
    let mut writer = encoder.write_header().unwrap();
    let data: Vec<u8> = td
        .framebuffer
        .iter()
        .flat_map(|row| row.iter())
        .flat_map(|c| {
            // data will be [RGBARGBA...]
            if *c {
                [0, 0, 0, 255]
            } else {
                [255, 255, 255, 255]
            }
        })
        .collect();
    writer.write_image_data(&data).unwrap();

    let tcp_data = gen_tcp_data(&td)?;

    if !dry_run {
        print_tcp_data(device_ip, &tcp_data)
    } else {
        analyze_tcp_data(&tcp_data)?;
        Ok(())
    }
}

pub fn do_print(args: PrintArgs) -> Result<()> {
    let device_ip = &args.printer;
    if let Some(tcp_data) = args.tcp_data {
        let label_data = fs::read(tcp_data).context("Failed to read TCP data")?;
        print_tcp_data(device_ip, &label_data)?;
    }
    if let Some(mac_addr) = args.mac_addr {
        print_mac_addr(&mac_addr, device_ip, args.dry_run)?;
    }
    if let Some(qr_text) = args.qr_text {
        print_qr_text(&qr_text, device_ip, args.dry_run)?;
    }
    Ok(())
}
