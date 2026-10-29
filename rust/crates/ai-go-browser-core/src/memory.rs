#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MemoryEntry {
    pub key: String,
    pub value: String,
    pub tags: Vec<String>,
}

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct MemoryStore {
    entries: Vec<MemoryEntry>,
}

impl MemoryStore {
    pub fn upsert(&mut self, entry: MemoryEntry) -> bool {
        if entry.key.trim().is_empty() {
            return false;
        }
        if let Some(existing) = self.entries.iter_mut().find(|e| e.key == entry.key) {
            *existing = entry;
        } else {
            self.entries.push(entry);
        }
        true
    }
    pub fn get(&self, key: &str) -> Option<&MemoryEntry> {
        self.entries.iter().find(|e| e.key == key)
    }
    pub fn search(&self, query: &str) -> Vec<&MemoryEntry> {
        let q = query.to_lowercase();
        self.entries
            .iter()
            .filter(|e| {
                e.key.to_lowercase().contains(&q)
                    || e.value.to_lowercase().contains(&q)
                    || e.tags.iter().any(|t| t.to_lowercase().contains(&q))
            })
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn upsert_replaces_key() {
        let mut m = MemoryStore::default();
        m.upsert(MemoryEntry {
            key: "x".into(),
            value: "1".into(),
            tags: vec![],
        });
        m.upsert(MemoryEntry {
            key: "x".into(),
            value: "2".into(),
            tags: vec![],
        });
        assert_eq!(m.get("x").unwrap().value, "2");
    }
}
