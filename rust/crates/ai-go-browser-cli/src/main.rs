use std::io::{self, BufRead, Write};
use ai_go_browser_core::{ActionRef, Browser, BrowserAction, HttpSnapshotBrowser, SafeHttpTransport};
use ai_go_browser_mcp::Server as MCPServer;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    let args: Vec<String> = std::env::args().collect();

    if args.len() > 1 && (args[1] == "--version" || args[1] == "-v") {
        println!("ai-rust-browser v1.0.0 (parity track)");
        return Ok(());
    }

    if args.len() > 1 && args[1] == "mcp" {
        // Run as standard Model Context Protocol (MCP) server
        let server = MCPServer::new();
        let stdin = io::stdin();
        let stdout = io::stdout();
        return server.serve(stdin.lock(), stdout.lock());
    }

    let transport = SafeHttpTransport::default();
    let mut browser = HttpSnapshotBrowser::new(transport);

    println!("AI Rust Browser REPL / CLI (Type 'help' or commands: open <url>, snapshot, click <ref>, fill <ref> <val>, text, quit)");
    let stdin = io::stdin();
    let mut reader = stdin.lock();
    let mut line = String::new();

    loop {
        print!("> ");
        io::stdout().flush()?;
        line.clear();
        if reader.read_line(&mut line)? == 0 {
            break;
        }

        let trimmed = line.trim();
        if trimmed.is_empty() {
            continue;
        }

        let parts: Vec<&str> = trimmed.split_whitespace().collect();
        let cmd = parts[0].to_lowercase();

        match cmd.as_str() {
            "quit" | "exit" => break,
            "help" => {
                println!("Comandos disponíveis:");
                println!("  open <url>               - Abre e renderiza semanticamente a página");
                println!("  snapshot                 - Mostra o snapshot de acessibilidade e ações");
                println!("  text                     - Extrai o texto limpo da página");
                println!("  click <ref>              - Clica no elemento (#ref ou l*/b*)");
                println!("  fill <ref> <valor>       - Preenche o campo de formulário");
                println!("  back                     - Volta no histórico");
                println!("  forward                  - Avança no histórico");
                println!("  mcp                      - Inicia o servidor MCP sobre stdin/stdout");
                println!("  quit                     - Sai do programa");
            }
            "open" => {
                if parts.len() < 2 {
                    eprintln!("Uso: open <url>");
                    continue;
                }
                let url = parts[1];
                match browser.navigate(url).await {
                    Ok(snap) => {
                        println!("\n=== {} ===", snap.title);
                        println!("URL: {}", snap.url);
                        println!("Elementos interativos detectados: {}", snap.elements.len());
                    }
                    Err(e) => eprintln!("Erro ao navegar: {:?}", e),
                }
            }
            "snapshot" => {
                if let Some(snap) = browser.current_snapshot() {
                    println!("\n# {}\nURL: {}\n", snap.title, snap.url);
                    for el in &snap.elements {
                        let ref_id = el.action.as_ref().map(|a| a.id.as_str()).unwrap_or("-");
                        let name = el.accessible_name().unwrap_or("elemento");
                        println!("- [{}] <{}> {}", ref_id, el.tag, name);
                    }
                } else {
                    println!("Nenhuma página carregada.");
                }
            }
            "text" => {
                match browser.perform(BrowserAction::ExtractText).await {
                    Ok(out) => {
                        println!("{}", out.text.unwrap_or_default());
                    }
                    Err(e) => eprintln!("Erro: {:?}", e),
                }
            }
            "click" => {
                if parts.len() < 2 {
                    eprintln!("Uso: click <ref>");
                    continue;
                }
                let r = parts[1];
                if let Some(aref) = ActionRef::new(r, r) {
                    match browser.perform(BrowserAction::Click { target: aref }).await {
                        Ok(_) => println!("Clicado com sucesso: {}", r),
                        Err(e) => eprintln!("Falha ao clicar: {:?}", e),
                    }
                } else {
                    eprintln!("Referência de ação inválida: {}", r);
                }
            }
            "fill" => {
                if parts.len() < 3 {
                    eprintln!("Uso: fill <ref> <valor>");
                    continue;
                }
                let r = parts[1];
                let val = parts[2..].join(" ");
                if let Some(aref) = ActionRef::new(r, r) {
                    match browser.perform(BrowserAction::Fill { target: aref, value: val.clone() }).await {
                        Ok(_) => println!("Preenchido {} com {:?}", r, val),
                        Err(e) => eprintln!("Falha ao preencher: {:?}", e),
                    }
                } else {
                    eprintln!("Referência de ação inválida: {}", r);
                }
            }
            "back" => {
                match browser.perform(BrowserAction::Back).await {
                    Ok(_) => println!("Voltou no histórico."),
                    Err(e) => eprintln!("Erro: {:?}", e),
                }
            }
            "forward" => {
                match browser.perform(BrowserAction::Forward).await {
                    Ok(_) => println!("Avançou no histórico."),
                    Err(e) => eprintln!("Erro: {:?}", e),
                }
            }
            _ => {
                if cmd.starts_with("http://") || cmd.starts_with("https://") {
                    match browser.navigate(&cmd).await {
                        Ok(snap) => {
                            println!("\n=== {} ===", snap.title);
                            println!("URL: {}", snap.url);
                        }
                        Err(e) => eprintln!("Erro ao navegar: {:?}", e),
                    }
                } else {
                    eprintln!("Comando desconhecido: '{}'. Digite 'help' para ajuda.", cmd);
                }
            }
        }
    }

    Ok(())
}
