import React, { useState, useEffect } from 'react';
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

function App() {
  const [selected, setSelected] = useState(0);
  return <FileTree selectedIndex={selected} />;
}

// Just render once for startup benchmark
const { unmount } = render(<App />);
setTimeout(() => {
  unmount();
  process.exit(0);
}, 100);
