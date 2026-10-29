pub mod types;
pub mod storage;
pub mod engine;

pub use types::{ActionFeedback, DomainPlaybook, HandoffSession, PageRecord};
pub use storage::{Storage, sanitize_fts5_query};
pub use engine::Engine;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_memory_engine_learn_and_search() {
        let engine = Engine::in_memory().expect("in memory engine");

        // 1. Learn selector
        engine.learn_selector("testsite.com", "login_btn", "#submit").unwrap();
        let sel = engine.get_learned_selector("testsite.com", "login_btn");
        assert_eq!(sel.as_deref(), Some("#submit"));

        // 2. Index page & search FTS5
        engine.record_visit(
            "https://testsite.com/docs",
            "Test Documentation",
            "High performance Go and Rust microservices architecture guide",
            200,
        ).unwrap();

        let hits = engine.search_memory("microservices architecture", 5).unwrap();
        assert!(!hits.is_empty());
        assert_eq!(hits[0].title, "Test Documentation");

        // 3. Export & resume handoff
        let handoff = engine.export_handoff("Test Goal", "https://testsite.com", "Title", vec!["step 1".into()]).unwrap();
        let resumed = engine.resume_handoff(&handoff.id).unwrap();
        assert!(resumed.is_some());
        assert_eq!(resumed.unwrap().goal, "Test Goal");
    }
}
