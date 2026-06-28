use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn version() -> String {
    "0.1.0".to_string()
}

#[cfg(test)]
mod tests {
    use super::version;

    #[test]
    fn version_is_set() {
        assert_eq!(version(), "0.1.0");
    }
}
