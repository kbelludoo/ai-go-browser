//! Differential HTTP parity against the Go reference.
//!
//! The example builds and starts the deterministic Go fixture server, replays
//! every scenario in `rust/parity/http-golden.json` through the Rust transport,
//! and compares three things:
//!
//! 1. the wire requests the server observed (method, path, query, headers,
//!    body and order),
//! 2. the final observation (status, URL, title, text, byte count, counts),
//! 3. the failure class for scenarios the reference rejects.
//!
//! Run from the repository root with:
//!
//! ```text
//! cargo run --manifest-path rust/Cargo.toml -p ai-go-browser-core --example http_parity
//! ```

mod golden;
mod replay;
mod server;

use std::{net::IpAddr, sync::Arc, time::Duration};

use ai_go_browser_core::{
    DestinationPolicy, MAX_BODY, MAX_REDIRECTS, SafeHttpTransport, SystemResolver, UrlPolicyError,
};
use golden::{Golden, compare_outcome, compare_requests};
use replay::replay;
use server::{fetch_records, repo_root, wait_for_server};
use url::Url;

#[tokio::main(flavor = "current_thread")]
async fn main() {
    if let Err(error) = run().await {
        eprintln!("http parity: {error}");
        std::process::exit(1);
    }
}

async fn run() -> Result<(), String> {
    let root = repo_root();
    let golden_path = root.join("rust/parity/http-golden.json");
    let golden: Golden = serde_json::from_slice(
        &std::fs::read(&golden_path)
            .map_err(|error| format!("read {}: {error}", golden_path.display()))?,
    )
    .map_err(|error| format!("parse {}: {error}", golden_path.display()))?;

    let _server = server::start_server(&root)?;
    let info_path = root.join("rust/parity/http-server.json");
    let base = wait_for_server(&info_path).await?;

    let mut differences = 0;
    let mut mismatched = 0;
    for scenario in &golden.scenarios {
        // The Go generator builds a new Browser per scenario, so cookies from
        // one scenario never reach the next one.
        let transport = SafeHttpTransport::with_adapters(
            Duration::from_secs(20),
            MAX_BODY,
            MAX_REDIRECTS,
            Arc::new(Loopback),
            Arc::new(SystemResolver),
        );
        let mut diffs = Vec::new();
        let observed = replay(&transport, &base, scenario).await;
        compare_outcome(&scenario.outcome, &observed, &mut diffs);
        match fetch_records(&base, &scenario.name).await {
            Ok(records) => compare_requests(&scenario.requests, &records, &mut diffs),
            Err(error) => diffs.push(format!("request log: {error}")),
        }
        if diffs.is_empty() {
            println!("{:<16} match", scenario.name);
        } else {
            mismatched += 1;
            differences += diffs.len();
            println!("{:<16} {} diffs", scenario.name, diffs.len());
            for diff in diffs {
                println!("  {diff}");
            }
        }
    }

    println!(
        "{}/{} scenarios match, {} differences",
        golden.scenarios.len() - mismatched,
        golden.scenarios.len(),
        differences
    );
    if differences == 0 {
        Ok(())
    } else {
        Err(format!("{differences} differences"))
    }
}

/// The example talks only to the loopback fixture server. The transport still
/// applies this policy to every hop and every resolved address.
struct Loopback;

impl DestinationPolicy for Loopback {
    fn check_url(&self, url: &Url) -> Result<(), UrlPolicyError> {
        if url.host_str() != Some("127.0.0.1") {
            return Err(UrlPolicyError::LocalHost);
        }
        Ok(())
    }

    fn check_address(&self, address: IpAddr) -> Result<(), UrlPolicyError> {
        if address.is_loopback() {
            Ok(())
        } else {
            Err(UrlPolicyError::PrivateAddress)
        }
    }
}
