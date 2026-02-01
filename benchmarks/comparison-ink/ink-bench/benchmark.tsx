import React, { useState, useEffect, useRef } from 'react';
import { render, Box, Text } from 'ink';

interface FileTreeProps {
  selectedIndex: number;
}

function FileTree({ selectedIndex }: FileTreeProps) {
  const items = [];
  for (let i = 0; i < 100; i++) {
    const isSelected = i === selectedIndex;
    const prefix = isSelected ? '> ' : '  ';
    items.push(
      <Text key={i} color={isSelected ? 'cyan' : 'white'} bold={isSelected}>
        {prefix}├── file-{String(i).padStart(3, '0')}.go
      </Text>
    );
  }

  return (
    <Box flexDirection="column" borderStyle="round" padding={1}>
      <Text color="green" bold>File Browser (ink)</Text>
      <Box height={1} />
      {items}
    </Box>
  );
}

const mode = process.argv[2] || 'benchmark';

async function measureStartup() {
  const start = performance.now();

  const App = () => <FileTree selectedIndex={0} />;
  const { unmount } = render(<App />, { stdout: process.stderr }); // Redirect to avoid output

  const elapsed = performance.now() - start;
  unmount();

  console.log(`Startup time: ${elapsed.toFixed(2)}ms`);
}

async function measureMemory() {
  const before = process.memoryUsage();

  const App = () => <FileTree selectedIndex={0} />;
  const { unmount } = render(<App />, { stdout: process.stderr });

  // Force GC if available
  if (global.gc) global.gc();

  const after = process.memoryUsage();
  unmount();

  const heapUsed = (after.heapUsed - before.heapUsed) / (1024 * 1024);
  console.log(`Memory used: ${heapUsed.toFixed(2)} MB`);
}

async function measureIdleCPU() {
  const App = () => <FileTree selectedIndex={0} />;
  const { unmount } = render(<App />, { stdout: process.stderr });

  const startUsage = process.cpuUsage();
  const startTime = performance.now();

  await new Promise(resolve => setTimeout(resolve, 2000));

  const endUsage = process.cpuUsage(startUsage);
  const elapsed = performance.now() - startTime;

  unmount();

  const cpuTime = (endUsage.user + endUsage.system) / 1000; // Convert to ms
  const cpuPercent = (cpuTime / elapsed) * 100;

  console.log(`Idle CPU: ${cpuPercent.toFixed(2)}%`);
}

async function measureUpdates() {
  let setSelected: (n: number) => void;

  function App() {
    const [selected, _setSelected] = useState(0);
    setSelected = _setSelected;
    return <FileTree selectedIndex={selected} />;
  }

  const { unmount } = render(<App />, { stdout: process.stderr });

  // Wait for initial render
  await new Promise(resolve => setTimeout(resolve, 100));

  const start = performance.now();
  for (let i = 0; i < 1000; i++) {
    setSelected(i % 100);
    // React batches updates, we need to wait for them
    await new Promise(resolve => setImmediate(resolve));
  }
  const elapsed = performance.now() - start;

  unmount();

  console.log(`1000 updates: ${elapsed.toFixed(0)}ms (${(1000 / elapsed * 1000).toFixed(0)} updates/sec)`);
}

async function runAllBenchmarks() {
  console.log('=== Ink Benchmark ===');
  console.log(`Node version: ${process.version}\n`);

  await measureStartup();
  await measureMemory();
  await measureIdleCPU();
  await measureUpdates();
}

switch (mode) {
  case 'startup':
    measureStartup();
    break;
  case 'memory':
    measureMemory();
    break;
  case 'idle':
    measureIdleCPU();
    break;
  case 'updates':
    measureUpdates();
    break;
  case 'benchmark':
    runAllBenchmarks();
    break;
  default:
    console.log('Usage: bun benchmark.ts [startup|memory|idle|updates|benchmark]');
}
