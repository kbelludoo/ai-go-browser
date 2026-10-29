//! Loopback HTTP fixture server and transport adapter doubles.
//!
//! The fixture server binds `127.0.0.1:0`, counts every accepted connection, and
//! lets each test decide the response. Counting accepts is how the tests prove
//! that a rejected destination never received a request at all.

#![forbid(unsafe_code)]
#![allow(dead_code)]

use std::{
    io::{Read, Write},
    net::{IpAddr, Shutdown, SocketAddr, TcpListener, TcpStream},
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicUsize, Ordering},
    },
    thread::{self, JoinHandle},
    time::Duration,
};

use ai_go_browser_core::{
    DestinationPolicy, HostResolver, PublicInternet, TransportError, UrlPolicyError,
};
use async_trait::async_trait;
use url::Url;

type Route = dyn Fn(&str, &mut TcpStream) + Send + Sync + 'static;

/// A loopback HTTP server that records how many connections it accepted.
pub struct TestServer {
    origin: String,
    port: u16,
    hits: Arc<AtomicUsize>,
    stop: Arc<AtomicBool>,
    join: Option<JoinHandle<()>>,
}

impl TestServer {
    pub fn spawn(route: impl Fn(&str, &mut TcpStream) + Send + Sync + 'static) -> Self {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind loopback fixture");
        listener
            .set_nonblocking(true)
            .expect("nonblocking fixture listener");
        let address = listener.local_addr().expect("fixture address");
        let hits = Arc::new(AtomicUsize::new(0));
        let stop = Arc::new(AtomicBool::new(false));
        let route: Arc<Route> = Arc::new(route);

        let accept_hits = Arc::clone(&hits);
        let accept_stop = Arc::clone(&stop);
        let join = thread::spawn(move || {
            while !accept_stop.load(Ordering::SeqCst) {
                match listener.accept() {
                    Ok((mut stream, _)) => {
                        accept_hits.fetch_add(1, Ordering::SeqCst);
                        let path = request_path(&read_request(&mut stream));
                        let route = Arc::clone(&route);
                        thread::spawn(move || route(&path, &mut stream));
                    }
                    Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {
                        thread::sleep(Duration::from_millis(5));
                    }
                    Err(_) => break,
                }
            }
        });

        Self {
            origin: format!("http://{address}"),
            port: address.port(),
            hits,
            stop,
            join: Some(join),
        }
    }

    pub fn origin(&self) -> &str {
        &self.origin
    }

    pub fn port(&self) -> u16 {
        self.port
    }

    /// Connections accepted so far, including ones that never sent a request.
    pub fn hits(&self) -> usize {
        self.hits.load(Ordering::SeqCst)
    }
}

impl Drop for TestServer {
    fn drop(&mut self) {
        self.stop.store(true, Ordering::SeqCst);
        if let Some(join) = self.join.take() {
            let _ = join.join();
        }
    }
}

fn read_request(stream: &mut TcpStream) -> String {
    let _ = stream.set_read_timeout(Some(Duration::from_secs(2)));
    let mut buffer = Vec::new();
    let mut chunk = [0_u8; 1024];
    loop {
        match stream.read(&mut chunk) {
            Ok(0) | Err(_) => break,
            Ok(read) => {
                buffer.extend_from_slice(&chunk[..read]);
                if buffer.windows(4).any(|window| window == b"\r\n\r\n") {
                    break;
                }
            }
        }
    }
    String::from_utf8_lossy(&buffer).into_owned()
}

fn request_path(request: &str) -> String {
    request
        .lines()
        .next()
        .and_then(|line| line.split_whitespace().nth(1))
        .unwrap_or("/")
        .to_string()
}

pub fn send_response(stream: &mut TcpStream, status: &str, headers: &[(&str, &str)], body: &[u8]) {
    let mut head = format!(
        "HTTP/1.1 {status}\r\nContent-Length: {}\r\nConnection: close\r\n",
        body.len()
    );
    for (name, value) in headers {
        head.push_str(&format!("{name}: {value}\r\n"));
    }
    head.push_str("\r\n");
    let _ = stream.write_all(head.as_bytes());
    let _ = stream.write_all(body);
    finish(stream);
}

pub fn send_redirect(stream: &mut TcpStream, location: &str) {
    send_response(stream, "302 Found", &[("Location", location)], b"");
}

/// Stream a body with `Transfer-Encoding: chunked` and no content length, so a
/// size cap can only be enforced while reading.
pub fn send_chunked(stream: &mut TcpStream, chunks: &[&[u8]]) {
    let _ = stream
        .write_all(b"HTTP/1.1 200 OK\r\nConnection: close\r\nTransfer-Encoding: chunked\r\n\r\n");
    for chunk in chunks {
        let _ = stream.write_all(format!("{:x}\r\n", chunk.len()).as_bytes());
        let _ = stream.write_all(chunk);
        let _ = stream.write_all(b"\r\n");
    }
    let _ = stream.write_all(b"0\r\n\r\n");
    finish(stream);
}

fn finish(stream: &mut TcpStream) {
    let _ = stream.flush();
    let _ = stream.shutdown(Shutdown::Write);
}

/// Allow exactly one origin (the fixture) and delegate everything else to the
/// production policy, so redirects to other destinations stay blocked.
pub struct AllowOrigin {
    origin: String,
}

impl AllowOrigin {
    pub fn new(origin: impl Into<String>) -> Arc<Self> {
        Arc::new(Self {
            origin: origin.into(),
        })
    }
}

impl DestinationPolicy for AllowOrigin {
    fn check_url(&self, url: &Url) -> Result<(), UrlPolicyError> {
        if url.origin().ascii_serialization() == self.origin {
            Ok(())
        } else {
            PublicInternet.check_url(url)
        }
    }

    fn check_address(&self, address: IpAddr) -> Result<(), UrlPolicyError> {
        if address.is_loopback() {
            Ok(())
        } else {
            PublicInternet.check_address(address)
        }
    }
}

/// Resolver that always returns the same answers and counts every lookup.
pub struct StaticResolver {
    answers: Vec<SocketAddr>,
    calls: Arc<AtomicUsize>,
}

impl StaticResolver {
    pub fn new(answers: Vec<SocketAddr>) -> Self {
        Self {
            answers,
            calls: Arc::new(AtomicUsize::new(0)),
        }
    }

    pub fn calls(&self) -> usize {
        self.calls.load(Ordering::SeqCst)
    }
}

#[async_trait]
impl HostResolver for StaticResolver {
    async fn resolve(&self, _host: &str, port: u16) -> Result<Vec<SocketAddr>, TransportError> {
        self.calls.fetch_add(1, Ordering::SeqCst);
        Ok(self
            .answers
            .iter()
            .map(|answer| SocketAddr::new(answer.ip(), port))
            .collect())
    }
}

/// Resolver that answers once with a validated address and would answer with a
/// different (private) address on a second lookup, modelling DNS rebinding.
pub struct RebindingResolver {
    first: Vec<SocketAddr>,
    later: Vec<SocketAddr>,
    calls: Arc<AtomicUsize>,
}

impl RebindingResolver {
    pub fn new(first: Vec<SocketAddr>, later: Vec<SocketAddr>) -> Self {
        Self {
            first,
            later,
            calls: Arc::new(AtomicUsize::new(0)),
        }
    }

    pub fn calls(&self) -> usize {
        self.calls.load(Ordering::SeqCst)
    }
}

#[async_trait]
impl HostResolver for RebindingResolver {
    async fn resolve(&self, _host: &str, port: u16) -> Result<Vec<SocketAddr>, TransportError> {
        let answers = if self.calls.fetch_add(1, Ordering::SeqCst) == 0 {
            &self.first
        } else {
            &self.later
        };
        Ok(answers
            .iter()
            .map(|answer| SocketAddr::new(answer.ip(), port))
            .collect())
    }
}
