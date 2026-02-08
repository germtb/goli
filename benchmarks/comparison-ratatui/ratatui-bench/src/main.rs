use ratatui::{
    backend::CrosstermBackend,
    layout::Rect,
    style::{Color, Modifier, Style},
    text::{Line, Span},
    widgets::{Block, Borders, Paragraph},
    Frame, Terminal,
};
use std::{
    env,
    time::{Duration, Instant},
};

const ITEM_COUNT: usize = 100;

fn render_file_tree(frame: &mut Frame, selected_index: usize) {
    let area = frame.size();

    let block = Block::default()
        .borders(Borders::ALL)
        .border_type(ratatui::widgets::BorderType::Rounded)
        .title(" File Browser (ratatui) ")
        .title_style(Style::default().fg(Color::Green).add_modifier(Modifier::BOLD));

    let inner = block.inner(area);
    frame.render_widget(block, area);

    let mut lines: Vec<Line> = Vec::with_capacity(ITEM_COUNT + 2);
    lines.push(Line::from(""));

    for i in 0..ITEM_COUNT {
        let is_selected = i == selected_index;
        let prefix = if is_selected { "> " } else { "  " };
        let text = format!("{}├── file-{:03}.go", prefix, i);

        let style = if is_selected {
            Style::default()
                .fg(Color::Cyan)
                .add_modifier(Modifier::BOLD)
        } else {
            Style::default().fg(Color::White)
        };

        lines.push(Line::from(Span::styled(text, style)));
    }

    let paragraph = Paragraph::new(lines);
    frame.render_widget(paragraph, inner);
}

fn render_large_grid(frame: &mut Frame, rows: usize, cols: usize, highlight: usize) {
    let area = frame.size();
    let mut lines: Vec<Line> = Vec::with_capacity(rows);

    for r in 0..rows {
        let mut spans: Vec<Span> = Vec::with_capacity(cols);
        for c in 0..cols {
            let idx = r * cols + c;
            let (ch, style) = if idx == highlight {
                (
                    "█",
                    Style::default()
                        .fg(Color::Cyan)
                        .add_modifier(Modifier::BOLD),
                )
            } else {
                ("·", Style::default().fg(Color::White))
            };
            spans.push(Span::styled(ch, style));
        }
        lines.push(Line::from(spans));
    }

    let paragraph = Paragraph::new(lines);
    frame.render_widget(paragraph, area);
}

fn measure_startup() {
    let start = Instant::now();

    // Create a dummy buffer backend to avoid terminal manipulation
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    let mut terminal = Terminal::new(backend).unwrap();

    terminal
        .draw(|frame| {
            render_file_tree(frame, 0);
        })
        .unwrap();

    let elapsed = start.elapsed();
    println!("Startup time: {:.2}ms", elapsed.as_secs_f64() * 1000.0);
}

fn measure_memory() {
    // Get memory before
    let before = get_memory_usage();

    // Create terminal and render
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    let mut terminal = Terminal::new(backend).unwrap();

    terminal
        .draw(|frame| {
            render_file_tree(frame, 0);
        })
        .unwrap();

    // Get memory after
    let after = get_memory_usage();

    let used_mb = (after.saturating_sub(before)) as f64 / (1024.0 * 1024.0);
    println!("Memory used: {:.2} MB", used_mb);
}

#[cfg(target_os = "macos")]
fn get_memory_usage() -> usize {
    use std::mem::MaybeUninit;

    #[repr(C)]
    struct TaskBasicInfo {
        virtual_size: u64,
        resident_size: u64,
        resident_size_max: u64,
        user_time: u64,
        system_time: u64,
        policy: i32,
        suspend_count: i32,
    }

    extern "C" {
        fn mach_task_self() -> u32;
        fn task_info(
            target_task: u32,
            flavor: i32,
            task_info_out: *mut TaskBasicInfo,
            task_info_count: *mut u32,
        ) -> i32;
    }

    const TASK_BASIC_INFO: i32 = 5;
    const TASK_BASIC_INFO_COUNT: u32 = 10;

    unsafe {
        let mut info = MaybeUninit::<TaskBasicInfo>::uninit();
        let mut count = TASK_BASIC_INFO_COUNT;
        let result = task_info(
            mach_task_self(),
            TASK_BASIC_INFO,
            info.as_mut_ptr(),
            &mut count,
        );
        if result == 0 {
            info.assume_init().resident_size as usize
        } else {
            0
        }
    }
}

#[cfg(target_os = "linux")]
fn get_memory_usage() -> usize {
    use std::fs;
    if let Ok(status) = fs::read_to_string("/proc/self/status") {
        for line in status.lines() {
            if line.starts_with("VmRSS:") {
                if let Some(kb) = line.split_whitespace().nth(1) {
                    if let Ok(kb) = kb.parse::<usize>() {
                        return kb * 1024;
                    }
                }
            }
        }
    }
    0
}

#[cfg(not(any(target_os = "macos", target_os = "linux")))]
fn get_memory_usage() -> usize {
    0
}

#[cfg(unix)]
fn get_cpu_time() -> Duration {
    use std::mem::MaybeUninit;

    #[repr(C)]
    struct Timeval {
        tv_sec: i64,
        tv_usec: i32,
    }

    #[repr(C)]
    struct Rusage {
        ru_utime: Timeval,
        ru_stime: Timeval,
        _padding: [u8; 128], // padding for other fields
    }

    extern "C" {
        fn getrusage(who: i32, usage: *mut Rusage) -> i32;
    }

    const RUSAGE_SELF: i32 = 0;

    unsafe {
        let mut usage = MaybeUninit::<Rusage>::uninit();
        if getrusage(RUSAGE_SELF, usage.as_mut_ptr()) == 0 {
            let usage = usage.assume_init();
            let user = Duration::new(
                usage.ru_utime.tv_sec as u64,
                usage.ru_utime.tv_usec as u32 * 1000,
            );
            let sys = Duration::new(
                usage.ru_stime.tv_sec as u64,
                usage.ru_stime.tv_usec as u32 * 1000,
            );
            user + sys
        } else {
            Duration::ZERO
        }
    }
}

#[cfg(not(unix))]
fn get_cpu_time() -> Duration {
    Duration::ZERO
}

fn measure_idle_cpu() {
    // Create terminal
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    let mut terminal = Terminal::new(backend).unwrap();

    // Initial render
    terminal
        .draw(|frame| {
            render_file_tree(frame, 0);
        })
        .unwrap();

    // Measure CPU over 2 seconds idle
    let cpu_start = get_cpu_time();
    let start = Instant::now();

    std::thread::sleep(Duration::from_secs(2));

    let cpu_end = get_cpu_time();
    let elapsed = start.elapsed();

    let cpu_used = cpu_end.saturating_sub(cpu_start);
    let cpu_percent = (cpu_used.as_secs_f64() / elapsed.as_secs_f64()) * 100.0;

    println!("Idle CPU: {:.2}%", cpu_percent);
}

fn measure_updates() {
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    // Use fixed viewport to match goli benchmark (60x40)
    let mut terminal = Terminal::with_options(
        backend,
        ratatui::TerminalOptions {
            viewport: ratatui::Viewport::Fixed(Rect::new(0, 0, 60, 40)),
        },
    ).unwrap();

    // Measure 1000 updates
    let start = Instant::now();
    for i in 0..1000 {
        let selected = i % ITEM_COUNT;
        terminal
            .draw(|frame| {
                render_file_tree(frame, selected);
            })
            .unwrap();
    }
    let elapsed = start.elapsed();

    let updates_per_sec = 1000.0 / elapsed.as_secs_f64();
    println!(
        "1000 updates: {:.0}ms ({:.0} updates/sec)",
        elapsed.as_secs_f64() * 1000.0,
        updates_per_sec
    );
}

fn measure_fps() {
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    // Use fixed viewport to match goli benchmark (60x40)
    let mut terminal = Terminal::with_options(
        backend,
        ratatui::TerminalOptions {
            viewport: ratatui::Viewport::Fixed(Rect::new(0, 0, 60, 40)),
        },
    ).unwrap();

    // Measure frames over 1 second
    let mut render_count = 0;
    let mut selected = 0;
    let start = Instant::now();
    let deadline = start + Duration::from_secs(1);

    while Instant::now() < deadline {
        terminal
            .draw(|frame| {
                render_file_tree(frame, selected);
            })
            .unwrap();
        render_count += 1;
        selected = (selected + 1) % ITEM_COUNT;
    }

    let elapsed = start.elapsed();
    let fps = render_count as f64 / elapsed.as_secs_f64();

    println!("Max FPS: {:.0} (60x40 screen, 100 items)", fps);
}

fn measure_large_screen() {
    let rows = 50;
    let cols = 200;
    let total_cells = rows * cols;

    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    let mut terminal = Terminal::with_options(
        backend,
        ratatui::TerminalOptions {
            viewport: ratatui::Viewport::Fixed(Rect::new(0, 0, cols as u16, rows as u16)),
        },
    )
    .unwrap();

    // Measure frames over 1 second
    let mut render_count = 0;
    let mut highlight = 0;
    let start = Instant::now();
    let deadline = start + Duration::from_secs(1);

    while Instant::now() < deadline {
        terminal
            .draw(|frame| {
                render_large_grid(frame, rows, cols, highlight);
            })
            .unwrap();
        render_count += 1;
        highlight = (highlight + 1) % total_cells;
    }

    let elapsed = start.elapsed();
    let fps = render_count as f64 / elapsed.as_secs_f64();

    println!(
        "Large screen FPS: {:.0} ({}x{} = {} cells)",
        fps, cols, rows, total_cells
    );
}

fn run_all_benchmarks() {
    println!("=== ratatui Benchmark ===");
    println!("Rust version: {}\n", env!("CARGO_PKG_RUST_VERSION").is_empty().then(|| "stable").unwrap_or(env!("CARGO_PKG_RUST_VERSION")));

    measure_startup();
    measure_memory();
    measure_idle_cpu();
    measure_updates();
    measure_fps();
    measure_large_screen();
}

fn main() {
    let args: Vec<String> = env::args().collect();
    let mode = args.get(1).map(|s| s.as_str()).unwrap_or("benchmark");

    match mode {
        "startup" => measure_startup(),
        "memory" => measure_memory(),
        "idle" => measure_idle_cpu(),
        "updates" => measure_updates(),
        "fps" => measure_fps(),
        "large" => measure_large_screen(),
        "benchmark" => run_all_benchmarks(),
        "debug" => debug_sizes(),
        _ => println!("Usage: ratatui-bench [startup|memory|idle|updates|fps|large|benchmark|debug]"),
    }
}

fn debug_sizes() {
    // Test 1: Terminal::new() - what size does it use?
    let mut buffer = Vec::new();
    let backend = CrosstermBackend::new(&mut buffer);
    let mut terminal = Terminal::new(backend).unwrap();
    println!("Terminal::new() terminal.size(): {:?}", terminal.size());
    terminal.draw(|frame| {
        println!("Terminal::new() frame.size(): {:?}", frame.size());
    }).unwrap();

    // Test 2: Fixed viewport
    let mut buffer2 = Vec::new();
    let backend2 = CrosstermBackend::new(&mut buffer2);
    let mut terminal2 = Terminal::with_options(
        backend2,
        ratatui::TerminalOptions {
            viewport: ratatui::Viewport::Fixed(Rect::new(0, 0, 200, 50)),
        },
    ).unwrap();
    println!("Fixed viewport terminal.size(): {:?}", terminal2.size());
    terminal2.draw(|frame| {
        println!("Fixed viewport frame.size(): {:?}", frame.size());
    }).unwrap();
}
