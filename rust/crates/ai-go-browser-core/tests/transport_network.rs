//! End-to-end transport tests against a controlled loopback HTTP server.
//!
//! Unit tests cover the policy predicates in isolation. These tests exercise the
//! whole chain instead: URL validation, DNS resolution and pinning, connection,
//! redirect handling, timeouts, and the streaming size cap. Every rejection also
//! asserts the fixture server accepted no connection, which is the only way to
//! show that a forbidden request never left the process.

#![forbid(unsafe_code)]

mod support;

use std::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    sync::Arc,
    time::{Duration, Instant},
};

use ai_go_browser_core::{
    PublicInternet, SafeHttpTransport, SystemResolver, TransportError, UrlPolicyError,
};
use support::{
    AllowOrigin, RebindingResolver, StaticResolver, TestServer, send_chunked, send_redirect,
    send_response,
};

const CAP: usize = 1024;

fn loopback(port: u16) -> SocketAddr {
    SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), port)
}

fn transport_for(origin: &str, timeout: Duration, max_bytes: usize) -> SafeHttpTransport {
    SafeHttpTransport::with_adapters(
        timeout,
        max_bytes,
        5,
        AllowOrigin::new(origin),
        Arc::new(SystemResolver),
    )
}

#[tokio::test]
async fn private_literal_never_reaches_the_server() {
    let server = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"SECRET"));

    let error = SafeHttpTransport::default()
        .get(&format!("{}/secret", server.origin()))
        .await
        .unwrap_err();

    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::PrivateAddress)
    );
    assert_eq!(server.hits(), 0, "a blocked URL must not open a connection");
}

#[tokio::test]
async fn redirect_to_private_literal_is_blocked_before_connecting() {
    let target = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"STEAL"));
    let target_url = format!("{}/steal", target.origin());
    let redirector = TestServer::spawn(move |path, stream| {
        if path == "/start" {
            send_redirect(stream, &target_url);
        } else {
            send_response(stream, "200 OK", &[], b"unexpected")
        }
    });

    let error = transport_for(redirector.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/start", redirector.origin()))
        .await
        .unwrap_err();

    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::PrivateAddress)
    );
    assert_eq!(redirector.hits(), 1, "the allowed hop must be fetched");
    assert_eq!(target.hits(), 0, "the private redirect must never connect");
}

#[tokio::test]
async fn redirect_to_link_local_metadata_is_blocked() {
    let redirector = TestServer::spawn(|_, stream| {
        send_redirect(stream, "http://169.254.169.254/latest/meta-data/");
    });

    let error = transport_for(redirector.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/start", redirector.origin()))
        .await
        .unwrap_err();

    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::PrivateAddress)
    );
    assert_eq!(redirector.hits(), 1);
}

#[tokio::test]
async fn relative_redirect_is_followed_within_the_allowed_origin() {
    let server = TestServer::spawn(|path, stream| {
        if path == "/start" {
            send_redirect(stream, "/final");
        } else {
            send_response(stream, "200 OK", &[], b"final body");
        }
    });

    let response = transport_for(server.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/start", server.origin()))
        .await
        .expect("relative redirect should be followed");

    assert_eq!(response.status, 200);
    assert!(response.final_url.ends_with("/final"), "{response:?}");
    assert_eq!(response.body, b"final body");
    assert_eq!(server.hits(), 2, "both hops must be fetched");
}

#[tokio::test]
async fn resolved_answers_are_pinned_and_looked_up_once() {
    let server = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"pinned"));
    let origin = format!("http://pinned.test:{}", server.port());
    let resolver = Arc::new(StaticResolver::new(vec![loopback(server.port())]));

    let transport = SafeHttpTransport::with_adapters(
        Duration::from_secs(5),
        CAP,
        5,
        AllowOrigin::new(origin.clone()),
        Arc::clone(&resolver) as Arc<dyn ai_go_browser_core::HostResolver>,
    );

    let response = transport.get(&origin).await.expect("pinned fetch");

    assert_eq!(response.body, b"pinned");
    assert_eq!(resolver.calls(), 1, "exactly one lookup per hop");
    assert_eq!(server.hits(), 1, "the pinned answer must be used");
}

#[tokio::test]
async fn rebinding_answer_is_never_used() {
    let server = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"first"));
    let origin = format!("http://rebind.test:{}", server.port());
    let resolver = Arc::new(RebindingResolver::new(
        vec![loopback(server.port())],
        vec![SocketAddr::new(
            IpAddr::V4(Ipv4Addr::new(10, 0, 0, 1)),
            server.port(),
        )],
    ));

    let transport = SafeHttpTransport::with_adapters(
        Duration::from_secs(5),
        CAP,
        5,
        AllowOrigin::new(origin.clone()),
        Arc::clone(&resolver) as Arc<dyn ai_go_browser_core::HostResolver>,
    );

    let response = transport.get(&origin).await.expect("first answer is used");

    assert_eq!(response.body, b"first");
    assert_eq!(resolver.calls(), 1, "no second lookup may happen");
    assert_eq!(
        server.hits(),
        1,
        "the connection stays on the validated answer"
    );
}

#[tokio::test]
async fn hostname_with_private_answer_never_connects() {
    let server = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"SECRET"));
    let resolver = Arc::new(StaticResolver::new(vec![loopback(server.port())]));
    let transport = SafeHttpTransport::with_adapters(
        Duration::from_secs(5),
        CAP,
        5,
        Arc::new(PublicInternet),
        Arc::clone(&resolver) as Arc<dyn ai_go_browser_core::HostResolver>,
    );

    let error = transport.get("http://internal.test/").await.unwrap_err();

    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::PrivateAddress)
    );
    assert_eq!(resolver.calls(), 1, "the answer must be checked");
    assert_eq!(server.hits(), 0, "the rejected answer must not be dialled");
}

#[tokio::test]
async fn mixed_answers_fail_closed_without_connecting() {
    let server = TestServer::spawn(|_, stream| send_response(stream, "200 OK", &[], b"SECRET"));
    let origin = format!("http://mixed.test:{}", server.port());
    let resolver = Arc::new(StaticResolver::new(vec![
        loopback(server.port()),
        SocketAddr::new(IpAddr::V4(Ipv4Addr::new(10, 0, 0, 1)), server.port()),
    ]));

    let transport = SafeHttpTransport::with_adapters(
        Duration::from_secs(5),
        CAP,
        5,
        AllowOrigin::new(origin.clone()),
        Arc::clone(&resolver) as Arc<dyn ai_go_browser_core::HostResolver>,
    );

    let error = transport.get(&origin).await.unwrap_err();

    assert_eq!(
        error,
        TransportError::Policy(UrlPolicyError::PrivateAddress)
    );
    assert_eq!(resolver.calls(), 1);
    assert_eq!(server.hits(), 0, "one rejected answer must fail the hop");
}

#[tokio::test]
async fn silent_server_times_out() {
    let server = TestServer::spawn(|_, stream| {
        std::thread::sleep(Duration::from_millis(900));
        send_response(stream, "200 OK", &[], b"late");
    });

    let started = Instant::now();
    let error = transport_for(server.origin(), Duration::from_millis(250), CAP)
        .get(&format!("{}/slow", server.origin()))
        .await
        .unwrap_err();
    let elapsed = started.elapsed();

    assert!(matches!(error, TransportError::Request(_)), "{error:?}");
    assert!(elapsed >= Duration::from_millis(200), "{elapsed:?}");
    assert!(elapsed < Duration::from_secs(5), "{elapsed:?}");
    assert_eq!(server.hits(), 1, "the request must have been sent");
}

#[tokio::test]
async fn declared_length_over_the_cap_is_rejected() {
    let server = TestServer::spawn(|_, stream| {
        send_response(stream, "200 OK", &[], &[b'x'; CAP * 4]);
    });

    let error = transport_for(server.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/big", server.origin()))
        .await
        .unwrap_err();

    assert_eq!(error, TransportError::ResponseTooLarge { limit: CAP });
    assert_eq!(server.hits(), 1);
}

#[tokio::test]
async fn streamed_body_over_the_cap_is_rejected() {
    let chunk = [b'x'; CAP / 2];
    let server = TestServer::spawn(move |_, stream| {
        send_chunked(stream, &[&chunk, &chunk, &chunk]);
    });

    let error = transport_for(server.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/stream", server.origin()))
        .await
        .unwrap_err();

    assert_eq!(error, TransportError::ResponseTooLarge { limit: CAP });
    assert_eq!(server.hits(), 1);
}

#[tokio::test]
async fn body_at_the_cap_is_accepted() {
    let chunk = [b'x'; CAP / 2];
    let server = TestServer::spawn(move |_, stream| {
        send_chunked(stream, &[&chunk, &chunk]);
    });

    let response = transport_for(server.origin(), Duration::from_secs(5), CAP)
        .get(&format!("{}/exact", server.origin()))
        .await
        .expect("a body exactly at the cap must be accepted");

    assert_eq!(response.status, 200);
    assert_eq!(response.body.len(), CAP);
}
