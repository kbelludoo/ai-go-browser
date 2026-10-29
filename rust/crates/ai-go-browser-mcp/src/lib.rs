pub mod server;

pub use server::{Server, JSONRPCRequest, JSONRPCResponse, ToolDefinition};

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;
    use serde_json::Value;

    #[tokio::test]
    async fn test_mcp_server_initialize_and_tools_list() {
        let server = Server::new();

        // 1. Initialize
        let init_req = "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\"}}\n";
        let mut out = Vec::new();
        server.serve(Cursor::new(init_req), &mut out).expect("serve init");

        let resp: Value = serde_json::from_slice(&out).expect("json resp");
        assert_eq!(resp["result"]["serverInfo"]["name"], "ai-rust-browser");

        // 2. Tools list
        let tools_req = "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n";
        let mut out = Vec::new();
        server.serve(Cursor::new(tools_req), &mut out).expect("serve tools");

        let resp: Value = serde_json::from_slice(&out).expect("json resp");
        let tools = resp["result"]["tools"].as_array().expect("tools array");
        assert_eq!(tools.len(), 10);
    }
}
