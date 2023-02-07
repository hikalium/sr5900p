#![feature(new_uninit)]
#![feature(slice_take)]
#![feature(exclusive_range_pattern)]

use anyhow::Result;
use argh::FromArgs;
use sr5900p::analyzer::analyze_tcp_data;
use sr5900p::print::do_print;
use sr5900p::print::PrintArgs;
use std::fs;

#[derive(FromArgs, PartialEq, Debug)]
/// Analyze the packet captures
#[argh(subcommand, name = "analyze")]
struct AnalyzeArgs {
    /// the raw dump of the TCP stream while printing
    #[argh(option)]
    tcp_data: String,
}
fn do_analyze(dump_file: &str) -> Result<()> {
    let data = fs::read(dump_file)?;
    analyze_tcp_data(&data)
}

#[derive(FromArgs, PartialEq, Debug)]
#[argh(subcommand)]
enum ArgsSubCommand {
    Analyze(AnalyzeArgs),
    Print(PrintArgs),
}
#[derive(Debug, FromArgs)]
/// Reach new heights.
struct Args {
    #[argh(subcommand)]
    nested: ArgsSubCommand,
}

fn main() -> Result<()> {
    let args: Args = argh::from_env();
    println!("{:?}", args);
    match args.nested {
        ArgsSubCommand::Analyze(args) => do_analyze(&args.tcp_data),
        ArgsSubCommand::Print(args) => do_print(&args),
    }
}
