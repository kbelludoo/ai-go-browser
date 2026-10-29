//! Fixture-server lifecycle and the request-log fetch.

use std::{
    path::{Path, PathBuf},
    process::{Child, Command, Stdio},
    time::{Duration, Instant},
};

use serde::Deserialize;

use crate::golden::RequestRecord;

pub struct Server {
    child: Child,
}

impl Drop for Server {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

pub fn repo_root() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .ancestors()
        .nth(3)
        .expect("crate lives three levels below the repository root")
        .to_path_buf()
}

pub fn start_server(root: &Path) -> Result<Server, String> {
    let binary = std::env::temp_dir().join(format!("aigb-httpgen{}", std::env::consts::EXE_SUFFIX));
    let status = Command::new("go")
        .args([
            "build",
            "-o",
            binary.to_str().ok_or("binary path is not UTF-8")?,
            "./rust/parity/httpgen",
        ])
        .current_dir(root)
        .status()
        .map_err(|error| format!("go build: {error}"))?;
    if !status.success() {
        return Err(format!("go build failed with {status}"));
    }
    let info = root.join("rust/parity/http-server.json");
    let _ = std::fs::remove_file(&info);
    let child = Command::new(&binary)
        .arg("-serve")
        .current_dir(root)
        .stdout(Stdio::null())
        .stderr(Stdio::inherit())
        .spawn()
        .map_err(|error| format!("start fixture server: {error}"))?;
    Ok(Server { child })
}

#[derive(Deserialize)]
struct ServerInfo {
    base_url: String,
}

pub async fn wait_for_server(path: &Path) -> Result<String, String> {
    let deadline = Instant::now() + Duration::from_secs(120);
    while Instant::now() < deadline {
        if let Ok(bytes) = std::fs::read(path)
            && let Ok(info) = serde_json::from_slice::<ServerInfo>(&bytes)
        {
            return Ok(info.base_url);
        }
        tokio::time::sleep(Duration::from_millis(100)).await;
    }
    Err(format!("fixture server did not write {}", path.display()))
}

pub async fn fetch_records(base: &str, scenario: &str) -> Result<Vec<RequestRecord>, String> {
    let url = format!("{base}/__requests?scenario={scenario}");
    let response = reqwest::get(url)
        .await
        .map_err(|error| error.to_string())?
        .error_for_status()
        .map_err(|error| error.to_string())?;
    response
        .json::<Option<Vec<RequestRecord>>>()
        .await
        .map(Option::unwrap_or_default)
        .map_err(|error| format!("decode request log: {error}"))
}
