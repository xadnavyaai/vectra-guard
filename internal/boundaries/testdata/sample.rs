// Sample Rust file with unsafe boundaries for boundaries tests.

pub fn ok_function() -> u32 {
    42
}

pub fn with_unsafe_block() {
    let mut x = 5;
    unsafe {
        let p = &mut x as *mut i32;
        *p = 10;
    }
}

pub unsafe fn raw_ptr_deref(p: *const u8) -> u8 {
    *p
}

unsafe impl Send for DangerousType {}

unsafe trait Pinky {
    fn dirty(&self);
}

pub struct DangerousType;
