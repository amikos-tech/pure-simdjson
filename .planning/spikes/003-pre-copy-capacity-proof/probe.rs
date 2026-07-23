#[derive(Clone, Debug, Default, Eq, PartialEq)]
struct Diagnostics {
    message: String,
    offset: Option<u64>,
}

#[derive(Debug, Default)]
struct Arena {
    bytes: Vec<u8>,
    padding_checks: usize,
    resize_calls: usize,
    copy_calls: usize,
    copied_bytes: usize,
}

#[derive(Debug)]
struct Parser {
    max_capacity: usize,
    arena: Arena,
    diagnostics: Diagnostics,
    reset_calls: usize,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Status {
    Ok,
    CapacityLimit,
    InvalidArgument,
}

impl Parser {
    fn new(max_capacity: usize) -> Self {
        Self {
            max_capacity,
            arena: Arena::default(),
            diagnostics: Diagnostics::default(),
            reset_calls: 0,
        }
    }

    fn with_stale_state(max_capacity: usize, arena_bytes: Vec<u8>) -> Self {
        Self {
            max_capacity,
            arena: Arena {
                bytes: arena_bytes,
                ..Arena::default()
            },
            diagnostics: Diagnostics {
                message: "stale syntax failure".to_owned(),
                offset: Some(7),
            },
            reset_calls: 0,
        }
    }

    fn clear_diagnostics(&mut self) {
        self.reset_calls += 1;
        self.diagnostics.message.clear();
        self.diagnostics.offset = None;
    }

    fn resize_and_copy(&mut self, input: &[u8], padding: usize) -> Result<(), Status> {
        self.arena.padding_checks += 1;
        let total_len = input
            .len()
            .checked_add(padding)
            .ok_or(Status::InvalidArgument)?;

        self.arena.resize_calls += 1;
        self.arena.bytes.resize(total_len, 0);

        self.arena.copy_calls += 1;
        self.arena.bytes[..input.len()].copy_from_slice(input);
        self.arena.copied_bytes += input.len();
        self.arena.bytes[input.len()..].fill(0);
        Ok(())
    }
}

fn parse_with_late_gate(parser: &mut Parser, input: &[u8], padding: usize) -> Status {
    parser.clear_diagnostics();
    if let Err(status) = parser.resize_and_copy(input, padding) {
        return status;
    }
    if input.len() > parser.max_capacity {
        return Status::CapacityLimit;
    }
    Status::Ok
}

fn parse_with_pre_copy_gate(parser: &mut Parser, input: &[u8], padding: usize) -> Status {
    parser.clear_diagnostics();
    if input.len() > parser.max_capacity {
        return Status::CapacityLimit;
    }
    if let Err(status) = parser.resize_and_copy(input, padding) {
        return status;
    }
    Status::Ok
}

fn main() {
    const PADDING: usize = 64;
    const EXACT_LIMIT: usize = 32;
    const LARGE_LIMIT: usize = 1024 * 1024;
    const LARGE_INPUT_LEN: usize = 8 * 1024 * 1024 + 1;

    let exact_input = vec![b'x'; EXACT_LIMIT];
    let mut exact_parser = Parser::new(EXACT_LIMIT);
    assert_eq!(
        parse_with_pre_copy_gate(&mut exact_parser, &exact_input, PADDING),
        Status::Ok
    );

    let rejected_input = vec![b'y'; EXACT_LIMIT + 1];
    let original_arena = vec![0xa5; 16];
    let mut rejected_parser = Parser::with_stale_state(EXACT_LIMIT, original_arena.clone());
    let original_len = rejected_parser.arena.bytes.len();
    let original_capacity = rejected_parser.arena.bytes.capacity();
    assert_eq!(
        parse_with_pre_copy_gate(&mut rejected_parser, &rejected_input, PADDING),
        Status::CapacityLimit
    );
    assert_eq!(rejected_parser.arena.bytes, original_arena);
    assert_eq!(rejected_parser.arena.bytes.len(), original_len);
    assert_eq!(rejected_parser.arena.bytes.capacity(), original_capacity);
    assert_eq!(rejected_parser.diagnostics, Diagnostics::default());

    let large_input = vec![b'z'; LARGE_INPUT_LEN];
    let mut late_parser = Parser::new(LARGE_LIMIT);
    let mut early_parser = Parser::new(LARGE_LIMIT);
    assert_eq!(
        parse_with_late_gate(&mut late_parser, &large_input, PADDING),
        Status::CapacityLimit
    );
    assert_eq!(
        parse_with_pre_copy_gate(&mut early_parser, &large_input, PADDING),
        Status::CapacityLimit
    );

    println!(
        concat!(
            "{{",
            "\"verified\":true,",
            "\"exact_limit\":{},",
            "\"exact_copied_bytes\":{},",
            "\"rejected_bytes\":{},",
            "\"rejected_padding_checks\":{},",
            "\"rejected_resize_calls\":{},",
            "\"rejected_copy_calls\":{},",
            "\"rejected_arena_unchanged\":true,",
            "\"rejected_diagnostics_cleared\":{},",
            "\"large_input_bytes\":{},",
            "\"late_gate_copied_bytes\":{},",
            "\"pre_copy_gate_copied_bytes\":{}",
            "}}"
        ),
        EXACT_LIMIT,
        exact_parser.arena.copied_bytes,
        rejected_input.len(),
        rejected_parser.arena.padding_checks,
        rejected_parser.arena.resize_calls,
        rejected_parser.arena.copy_calls,
        rejected_parser.diagnostics == Diagnostics::default(),
        LARGE_INPUT_LEN,
        late_parser.arena.copied_bytes,
        early_parser.arena.copied_bytes,
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    const PADDING: usize = 64;

    #[test]
    fn exact_limit_is_accepted_and_copied_once() {
        let input = vec![1; 32];
        let mut parser = Parser::new(32);

        assert_eq!(
            parse_with_pre_copy_gate(&mut parser, &input, PADDING),
            Status::Ok
        );
        assert_eq!(parser.arena.padding_checks, 1);
        assert_eq!(parser.arena.resize_calls, 1);
        assert_eq!(parser.arena.copy_calls, 1);
        assert_eq!(parser.arena.copied_bytes, input.len());
        assert_eq!(&parser.arena.bytes[..input.len()], input.as_slice());
    }

    #[test]
    fn oversized_input_is_rejected_before_padding_resize_or_copy() {
        let input = vec![2; 33];
        let original = vec![0xa5; 16];
        let mut parser = Parser::with_stale_state(32, original.clone());
        let original_len = parser.arena.bytes.len();
        let original_capacity = parser.arena.bytes.capacity();

        assert_eq!(
            parse_with_pre_copy_gate(&mut parser, &input, PADDING),
            Status::CapacityLimit
        );
        assert_eq!(parser.arena.padding_checks, 0);
        assert_eq!(parser.arena.resize_calls, 0);
        assert_eq!(parser.arena.copy_calls, 0);
        assert_eq!(parser.arena.copied_bytes, 0);
        assert_eq!(parser.arena.bytes, original);
        assert_eq!(parser.arena.bytes.len(), original_len);
        assert_eq!(parser.arena.bytes.capacity(), original_capacity);
    }

    #[test]
    fn capacity_rejection_clears_stale_diagnostics() {
        let mut parser = Parser::with_stale_state(32, vec![]);

        assert_eq!(
            parse_with_pre_copy_gate(&mut parser, &[3; 33], PADDING),
            Status::CapacityLimit
        );
        assert_eq!(parser.reset_calls, 1);
        assert_eq!(parser.diagnostics, Diagnostics::default());
    }

    #[test]
    fn pre_copy_gate_avoids_large_rejected_copy() {
        let input = vec![4; 8 * 1024 * 1024 + 1];
        let mut late_parser = Parser::new(1024 * 1024);
        let mut early_parser = Parser::new(1024 * 1024);

        assert_eq!(
            parse_with_late_gate(&mut late_parser, &input, PADDING),
            Status::CapacityLimit
        );
        assert_eq!(
            parse_with_pre_copy_gate(&mut early_parser, &input, PADDING),
            Status::CapacityLimit
        );
        assert_eq!(late_parser.arena.copied_bytes, input.len());
        assert_eq!(early_parser.arena.copied_bytes, 0);
        assert_eq!(early_parser.arena.resize_calls, 0);
    }
}
