use std::collections::BTreeMap;

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct Session {
    pub id: String,
    pub cookies: BTreeMap<String, String>,
    pub metadata: BTreeMap<String, String>,
}

impl Session {
    pub fn new(id: impl Into<String>) -> Option<Self> {
        let id = id.into();
        if id.trim().is_empty() {
            return None;
        }
        Some(Self {
            id,
            ..Self::default()
        })
    }
    pub fn set_cookie(&mut self, name: impl Into<String>, value: impl Into<String>) -> bool {
        let name = name.into();
        if name.trim().is_empty() {
            return false;
        }
        self.cookies.insert(name, value.into());
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn cookie_roundtrip_in_memory() {
        let mut s = Session::new("s1").unwrap();
        assert!(s.set_cookie("sid", "abc"));
        assert_eq!(s.cookies.get("sid").map(String::as_str), Some("abc"));
    }
}
