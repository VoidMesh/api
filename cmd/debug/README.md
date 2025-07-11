# VoidMesh Debug Tool

A comprehensive TUI debugging tool for the VoidMesh API built with Bubble Tea v2 and Lip Gloss.

## Features

### 🗺️ Chunk Explorer
- Interactive 16x16 grid visualization of chunks
- Real-time resource node display with colored symbols
- Navigate between chunks with arrow keys
- Detailed node information panel
- Auto-refresh functionality
- **Direct harvesting system** - harvest nodes instantly
- Visual feedback for harvested nodes
- Daily harvest limit tracking

### 👥 Player Activity Monitor *(Coming Soon)*
- Live monitoring of player activities
- Harvest action tracking
- Player statistics display
- Real-time updates

### 🗄️ Database Inspector *(Coming Soon)*
- Interactive SQL query interface
- Pre-built common queries
- Table browser
- Data export functionality

### ⚙️ Node Generator *(Coming Soon)*
- Create nodes with custom parameters
- Test spawn templates
- Bulk operations
- Spawn simulation

### 📊 System Overview *(Coming Soon)*
- Key performance metrics
- System health dashboard
- Activity charts
- Resource distribution analysis

## Usage

### Building
```bash
go build -o voidmesh-debug cmd/debug/main.go
```

### Running
```bash
# Start with default settings
./voidmesh-debug

# Specify database path
./voidmesh-debug --db=./path/to/game.db

# Start with specific view
./voidmesh-debug --view=chunks

# Enable debug logging
DEBUG=1 ./voidmesh-debug --log=debug
```

### Command Line Options
- `--db string`: Path to SQLite database (default: "./game.db")
- `--view string`: Starting view (menu, chunks, sessions, database, generator, overview) (default: "menu")
- `--log string`: Log level (debug, info, warn, error) (default: "info")

## Navigation

### Global Controls
- `q` or `Ctrl+C`: Quit (from menu) / Back to menu (from views)
- `?`: Toggle help screen
- `Tab`: Cycle through views
- `1-5`: Quick select view (from menu)

### Chunk Explorer
- `Arrow keys`: Move cursor within chunk
- `Shift+Arrow keys`: Navigate between chunks
- `r`: Refresh chunk data
- `a`: Toggle auto-refresh
- `i`: Toggle info panel
- `H`/`Enter`/`Space`: Harvest node at cursor
- **Direct harvesting**: Single action per node per day

### Symbols Legend

#### Resource Types
- `Fe` Iron Ore
- `Au` Gold Ore  
- `##` Wood
- `[]` Stone

#### Quality Levels
- `o` Poor Quality
- `O` Normal Quality
- `*` Rich Quality

#### Status Indicators
- `xx` Depleted
- `..` Respawning
- `><` Cursor Position
- Gray background: Harvested today

## Architecture

The debug tool follows a clean architecture with separate concerns:

```
cmd/debug/
├── main.go              # Entry point
├── models/              # View models (MVC pattern)
│   ├── app.go          # Main application controller
│   ├── menu.go         # Main menu
│   ├── chunk_explorer.go # Chunk visualization
│   ├── session_monitor.go # Session monitoring
│   ├── database.go     # Database inspector
│   ├── node_generator.go # Node creation
│   └── overview.go     # System dashboard
├── components/          # Reusable UI components
│   └── styles.go       # Lip Gloss styles and helpers
└── README.md           # This file
```

## Dependencies

- **Bubble Tea v2**: Modern TUI framework with enhanced keyboard support
- **Lip Gloss**: Terminal styling and layout library
- **Bubbles**: Pre-built UI components
- **Charmbracelet Log**: Structured logging

## Development

### Adding New Views
1. Create a new model in `models/`
2. Implement the `tea.Model` interface (Init, Update, View)
3. Add the view to the main app router
4. Update the menu with the new option

### Styling Guidelines
- Use predefined styles from `components/styles.go`
- Follow the established color scheme
- Ensure responsive design for different terminal sizes
- Test with both light and dark terminal themes

## Debugging

The debug tool automatically logs all activities to `debug.log` to prevent UI disruption. Enable different log levels:

```bash
# Info level (default)
./voidmesh-debug --log=info

# Debug level for detailed logging
./voidmesh-debug --log=debug

# Error level for minimal logging
./voidmesh-debug --log=error
```

Monitor the log file in real-time:
```bash
tail -f debug.log
```

## Recent Updates

### v2.0 - Direct Harvesting System
- **Breaking Change**: Removed harvest session system
- **New Feature**: Direct node harvesting with single API calls
- **Improvement**: Daily harvest limits per player per node
- **Enhancement**: Better UI feedback for harvest actions
- **Fix**: Improved log management to prevent UI disruption

## Future Enhancements

- [ ] Complete player activity monitor with real-time updates
- [ ] Full database inspector with query builder
- [ ] Node generator with form validation
- [ ] System overview with charts and metrics
- [ ] Export functionality for all views
- [ ] Configuration file support
- [ ] Plugin system for custom views
- [ ] Remote database support
- [ ] Performance profiling integration
- [ ] Tool and consumable support in harvest UI
- [ ] Character stats integration