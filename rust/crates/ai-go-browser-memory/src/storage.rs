use std::path::PathBuf;
use rusqlite::{params, Connection, Result as SqlResult};
use crate::types::{DomainPlaybook, HandoffSession, PageRecord};

pub struct Storage {
    conn: Connection,
}

impl Storage {
    pub fn open(db_path: Option<&str>) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let path = match db_path {
            Some(p) if !p.is_empty() => PathBuf::from(p),
            _ => PathBuf::from(".ai-browser-memory").join("browser_memory.db"),
        };

        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }

        let conn = Connection::open(&path)?;

        // Optimizations: WAL mode, busy timeout 5000, synchronous NORMAL
        conn.pragma_update(None, "journal_mode", "WAL")?;
        conn.pragma_update(None, "busy_timeout", 5000)?;
        conn.pragma_update(None, "synchronous", "NORMAL")?;

        let storage = Self { conn };
        storage.init_schema()?;
        Ok(storage)
    }

    pub fn in_memory() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let conn = Connection::open_in_memory()?;
        let storage = Self { conn };
        storage.init_schema()?;
        Ok(storage)
    }

    fn init_schema(&self) -> SqlResult<()> {
        self.conn.execute_batch(
            "
            CREATE TABLE IF NOT EXISTS playbooks (
                domain TEXT PRIMARY KEY,
                data_json TEXT NOT NULL,
                updated_at INTEGER NOT NULL
            );

            CREATE TABLE IF NOT EXISTS handoffs (
                id TEXT PRIMARY KEY,
                data_json TEXT NOT NULL,
                created_at INTEGER NOT NULL
            );

            CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
                url,
                title,
                domain,
                content,
                tokenize = 'unicode61 remove_diacritics 2'
            );

            CREATE TABLE IF NOT EXISTS page_bytecodes (
                url TEXT PRIMARY KEY,
                bytecode BLOB NOT NULL,
                updated_at INTEGER NOT NULL
            );
            ",
        )?;
        Ok(())
    }

    pub fn save_playbook(&mut self, pb: &mut DomainPlaybook) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        if pb.domain.is_empty() {
            return Err("domain cannot be empty".into());
        }
        pb.updated_at = chrono::Utc::now();
        let json_str = serde_json::to_string(pb)?;
        let updated_at_unix = pb.updated_at.timestamp();

        self.conn.execute(
            "INSERT INTO playbooks (domain, data_json, updated_at) VALUES (?1, ?2, ?3)
             ON CONFLICT(domain) DO UPDATE SET data_json = excluded.data_json, updated_at = excluded.updated_at",
            params![pb.domain, json_str, updated_at_unix],
        )?;
        Ok(())
    }

    pub fn get_playbook(&self, domain: &str) -> Result<Option<DomainPlaybook>, Box<dyn std::error::Error + Send + Sync>> {
        let mut stmt = self.conn.prepare("SELECT data_json FROM playbooks WHERE domain = ?1")?;
        let mut rows = stmt.query(params![domain])?;
        if let Some(row) = rows.next()? {
            let data_str: String = row.get(0)?;
            let pb: DomainPlaybook = serde_json::from_str(&data_str)?;
            Ok(Some(pb))
        } else {
            Ok(None)
        }
    }

    pub fn index_page(&mut self, rec: PageRecord) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        if rec.url.is_empty() {
            return Err("url cannot be empty".into());
        }

        let _ = self.conn.execute("DELETE FROM pages_fts WHERE url = ?1", params![rec.url]);

        let mut clean_content = rec.content.trim().to_string();
        if clean_content.len() > 200_000 {
            clean_content.truncate(200_000);
        }

        self.conn.execute(
            "INSERT INTO pages_fts (url, title, domain, content) VALUES (?1, ?2, ?3, ?4)",
            params![rec.url, rec.title, rec.domain, clean_content],
        )?;

        Ok(())
    }

    pub fn search_pages(&self, query: &str, limit: usize) -> Result<Vec<PageRecord>, Box<dyn std::error::Error + Send + Sync>> {
        let limit = if limit == 0 { 5 } else { limit };
        let sanitized = sanitize_fts5_query(query);
        if sanitized.is_empty() {
            return Ok(Vec::new());
        }

        let sql = "
            SELECT url, title, domain, snippet(pages_fts, 3, '<b>', '</b>', '...', 24), bm25(pages_fts) as rank
            FROM pages_fts
            WHERE pages_fts MATCH ?1
            ORDER BY rank ASC
            LIMIT ?2
        ";

        let mut stmt = self.conn.prepare(sql)?;
        let mut rows = stmt.query(params![sanitized, limit as i64])?;

        let mut results = Vec::new();
        while let Some(row) = rows.next()? {
            let url: String = row.get(0)?;
            let title: String = row.get(1)?;
            let domain: String = row.get(2)?;
            let snippet: String = row.get(3)?;
            let rank: f64 = row.get(4)?;

            results.push(PageRecord {
                url,
                title,
                domain,
                snippet,
                rank,
                ..Default::default()
            });
        }
        Ok(results)
    }

    pub fn save_handoff(&mut self, session: &HandoffSession) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let json_str = serde_json::to_string(session)?;
        let created_at = session.saved_at.timestamp();

        self.conn.execute(
            "INSERT INTO handoffs (id, data_json, created_at) VALUES (?1, ?2, ?3)
             ON CONFLICT(id) DO UPDATE SET data_json = excluded.data_json, created_at = excluded.created_at",
            params![session.id, json_str, created_at],
        )?;
        Ok(())
    }

    pub fn get_handoff(&self, id: &str) -> Result<Option<HandoffSession>, Box<dyn std::error::Error + Send + Sync>> {
        let mut stmt = self.conn.prepare("SELECT data_json FROM handoffs WHERE id = ?1")?;
        let mut rows = stmt.query(params![id])?;
        if let Some(row) = rows.next()? {
            let data_str: String = row.get(0)?;
            let h: HandoffSession = serde_json::from_str(&data_str)?;
            Ok(Some(h))
        } else {
            Ok(None)
        }
    }
}

pub fn sanitize_fts5_query(query: &str) -> String {
    let mut words = Vec::new();
    for token in query.split_whitespace() {
        let clean: String = token.chars().filter(|c| c.is_alphanumeric() || *c == '_' || *c == '-').collect();
        if !clean.is_empty() {
            words.push(format!("\"{clean}\""));
        }
    }
    words.join(" AND ")
}
