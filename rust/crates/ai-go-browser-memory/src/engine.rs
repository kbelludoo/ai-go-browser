use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use crate::storage::Storage;
use crate::types::{DomainPlaybook, HandoffSession, PageRecord};

pub struct Engine {
    storage: Arc<Mutex<Storage>>,
    cache: Arc<Mutex<HashMap<String, DomainPlaybook>>>,
}

impl Engine {
    pub fn new(db_path: Option<&str>) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let storage = Storage::open(db_path)?;
        let engine = Self {
            storage: Arc::new(Mutex::new(storage)),
            cache: Arc::new(Mutex::new(HashMap::new())),
        };
        engine.seed_universal_defaults()?;
        Ok(engine)
    }

    pub fn in_memory() -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let storage = Storage::in_memory()?;
        let engine = Self {
            storage: Arc::new(Mutex::new(storage)),
            cache: Arc::new(Mutex::new(HashMap::new())),
        };
        engine.seed_universal_defaults()?;
        Ok(engine)
    }

    fn seed_universal_defaults(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let defaults = vec![
            DomainPlaybook {
                domain: "github.com".into(),
                fast_paths: HashMap::from([
                    ("search".into(), "https://github.com/search?q={query}".into()),
                ]),
                known_selectors: HashMap::from([
                    ("search_input".into(), "input.header-search-input, input[name='q'], [data-target='qbsearch-input.inputButtonText']".into()),
                    ("sign_in".into(), "a[href*='/login']".into()),
                ]),
                dismiss_modals: vec![".js-cookie-consent-banner button".into()],
                ..Default::default()
            },
            DomainPlaybook {
                domain: "google.com".into(),
                fast_paths: HashMap::from([
                    ("search".into(), "https://www.google.com/search?q={query}".into()),
                ]),
                known_selectors: HashMap::from([
                    ("search_input".into(), "textarea[name='q'], input[name='q']".into()),
                    ("submit".into(), "input[name='btnK']".into()),
                ]),
                dismiss_modals: vec!["#L2AGLb".into(), "button[aria-label*='Concordo']".into(), "button[aria-label*='Aceitar']".into()],
                ..Default::default()
            },
        ];

        let mut storage = self.storage.lock().unwrap();
        let mut cache = self.cache.lock().unwrap();

        for mut pb in defaults {
            if storage.get_playbook(&pb.domain)?.is_none() {
                storage.save_playbook(&mut pb)?;
                cache.insert(pb.domain.clone(), pb);
            }
        }

        Ok(())
    }

    pub fn learn_selector(&self, domain: &str, key: &str, selector: &str) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut storage = self.storage.lock().unwrap();
        let mut cache = self.cache.lock().unwrap();

        let mut pb = match storage.get_playbook(domain)? {
            Some(existing) => existing,
            None => DomainPlaybook {
                domain: domain.to_string(),
                ..Default::default()
            },
        };

        pb.known_selectors.insert(key.to_string(), selector.to_string());
        storage.save_playbook(&mut pb)?;
        cache.insert(domain.to_string(), pb);
        Ok(())
    }

    pub fn get_learned_selector(&self, domain: &str, key: &str) -> Option<String> {
        let cache = self.cache.lock().unwrap();
        if let Some(pb) = cache.get(domain)
            && let Some(sel) = pb.known_selectors.get(key)
        {
            return Some(sel.clone());
        }
        drop(cache);

        let storage = self.storage.lock().unwrap();
        if let Ok(Some(pb)) = storage.get_playbook(domain) {
            pb.known_selectors.get(key).cloned()
        } else {
            None
        }
    }

    pub fn record_visit(&self, url: &str, title: &str, content: &str, status: i32) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let domain = url::Url::parse(url)
            .ok()
            .and_then(|u| u.host_str().map(|s| s.to_string()))
            .unwrap_or_default();

        let rec = PageRecord {
            url: url.to_string(),
            title: title.to_string(),
            domain,
            content: content.to_string(),
            status,
            visited_at: chrono::Utc::now(),
            ..Default::default()
        };

        let mut storage = self.storage.lock().unwrap();
        storage.index_page(rec)?;
        Ok(())
    }

    pub fn search_memory(&self, query: &str, limit: usize) -> Result<Vec<PageRecord>, Box<dyn std::error::Error + Send + Sync>> {
        let storage = self.storage.lock().unwrap();
        storage.search_pages(query, limit)
    }

    pub fn export_handoff(&self, goal: &str, url: &str, title: &str, steps: Vec<String>) -> Result<HandoffSession, Box<dyn std::error::Error + Send + Sync>> {
        let id = format!("{:x}", rand::random::<u64>());
        let session = HandoffSession {
            id,
            url: url.to_string(),
            title: title.to_string(),
            goal: goal.to_string(),
            steps_taken: steps,
            cookies: HashMap::new(),
            saved_at: chrono::Utc::now(),
        };

        let mut storage = self.storage.lock().unwrap();
        storage.save_handoff(&session)?;
        Ok(session)
    }

    pub fn resume_handoff(&self, id: &str) -> Result<Option<HandoffSession>, Box<dyn std::error::Error + Send + Sync>> {
        let storage = self.storage.lock().unwrap();
        storage.get_handoff(id)
    }
}
