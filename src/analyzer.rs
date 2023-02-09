use anyhow::anyhow;
use anyhow::Result;
use std::num::Wrapping;

pub fn analyze_tcp_data(data: &[u8]) -> Result<()> {
    println!("Size: {}", data.len());
    let mut i = 0;
    let mut num_data_rows = 0;
    while i < data.len() {
        match data[i] {
            0x1b => match data[i + 1] {
                0x7b => {
                    let payload_data = &data[i..i + 3 + data[i + 2] as usize];
                    println!("{payload_data:?}");
                    i += payload_data.len();
                    let mut payload_data = &payload_data[3..];

                    if payload_data.last().unwrap() != &0x7d {
                        return Err(anyhow!(
                            "Unexpected label data (not 0x7d): {:?}...",
                            &data[i..i + 16]
                        ));
                    }
                    payload_data.take_last();

                    if payload_data
                        .iter()
                        .map(|v| Wrapping(*v))
                        .sum::<Wrapping<u8>>()
                        .0
                        != payload_data.last().unwrap().wrapping_mul(2)
                    {
                        return Err(anyhow!(
                            "Unexpected label data (csum invalid): {:?}...",
                            &data[i..i + 16]
                        ));
                    }
                    // so the last byte of the payload_data is the checksum
                    payload_data.take_last();
                    if payload_data[0] == 76 {
                        let mut tape_len_bytes = [0u8; 4];
                        tape_len_bytes.copy_from_slice(&payload_data[1..5]);
                        let tape_len = u32::from_le_bytes(tape_len_bytes);
                        println!("cmd 0x1b 0x7b, {payload_data:?} tape_len = {}", tape_len);
                    } else {
                        println!("cmd 0x1b 0x7b, {payload_data:?}");
                    }
                }
                0x2e => {
                    if data[i + 2..i + 6] != [0, 0, 0, 1] {
                        return Err(anyhow!("Unexpected label data: {:?}...", &data[i..i + 16]));
                    }
                    let bits = data[i + 6] as usize + data[i + 7] as usize * 256;
                    let bytes = (bits + 7) / 8;
                    print!("cmd 0x1b 0x2e, bits = {bits}, bytes = {bytes}: ",);
                    let img_data = &data[i + 8..i + 8 + bytes];
                    for byte in img_data {
                        print!("{byte:08b}");
                    }
                    println!();
                    i += 8 + bytes;
                    num_data_rows += 1;
                }
                _ => {
                    return Err(anyhow!("Unexpected label data: {:?}...", &data[i..i + 16]));
                }
            },
            0x0c => {
                println!("cmd 0x0c (data end marker?)",);
                i += 1;
            }
            _ => {
                return Err(anyhow!("Unexpected label data: {:?}...", &data[i..]));
            }
        }
    }
    println!("num_data_rows = {}", num_data_rows);
    Ok(())
}
