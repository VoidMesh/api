# VoidMesh API Documentation

Welcome to the comprehensive documentation for VoidMesh API - a chunk-based resource harvesting system for multiplayer games.

## 📚 Documentation Overview

This documentation suite covers everything from quick start guides to deep architectural discussions. Choose your path based on your role and needs.

## 🚀 Getting Started

### New to VoidMesh?
1. **[Project Overview](project_overview.md)** - Understand the system design and philosophy
2. **[User Guide](user_guide.md)** - Learn how to integrate with the API
3. **[API Documentation](api_documentation.md)** - Complete REST API reference

### Want to Contribute?
1. **[Developer Guide](developer_guide.md)** - Setup, architecture, and development workflows
2. **[Implementation Guide](implementation_guide.md)** - Step-by-step implementation details

## 📖 Complete Documentation Set

### Core Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| **[API Documentation](api_documentation.md)** | Developers, Integrators | Complete REST API reference with request/response examples, error codes, and usage patterns |
| **[Developer Guide](developer_guide.md)** | Contributors, Maintainers | Architecture overview, development setup, testing strategies, and contribution guidelines |
| **[User Guide](user_guide.md)** | Game Developers, Integrators | Integration examples for JavaScript, Python, Unity C#, and game design considerations |

### Technical Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| **[Database Schema](database_schema_docs.md)** | Database Developers | Complete database design, relationships, indexes, and query patterns |
| **[Implementation Guide](implementation_guide.md)** | System Architects | Detailed implementation strategy and technical decisions |

### Design Documentation

| Document | Audience | Description |
|----------|----------|-------------|
| **[Project Overview](project_overview.md)** | Everyone | High-level system design, key principles, and architecture decisions |
| **[Game Design Summary](voidmesh_game_design_summary.md)** | Game Designers | Inspiration from EVE Online and GW2, game mechanics explanation |
| **[Design Discussion](design_discussion.md)** | Architects, Designers | Technical discussions and design rationale |

## 🎯 Documentation by Role

### 🎮 Game Developer
*"I want to integrate VoidMesh into my game"*

**Start Here:**
1. [User Guide](user_guide.md) - Integration examples and client libraries
2. [API Documentation](api_documentation.md) - Complete endpoint reference
3. [Project Overview](project_overview.md) - Understand the game mechanics

**Key Sections:**
- JavaScript/Unity integration examples
- Resource type explanations
- Error handling patterns
- Performance optimization tips

### 🔧 Backend Developer
*"I want to contribute to or modify VoidMesh"*

**Start Here:**
1. [Developer Guide](developer_guide.md) - Complete development environment setup
2. [Database Schema](database_schema_docs.md) - Understand data structures
3. [Implementation Guide](implementation_guide.md) - Technical implementation details

**Key Sections:**
- Go development patterns
- Database query optimization
- Testing strategies
- Security considerations

### 🏗️ System Architect
*"I want to understand the technical decisions and architecture"*

**Start Here:**
1. [Project Overview](project_overview.md) - High-level architecture
2. [Design Discussion](design_discussion.md) - Technical rationale
3. [Implementation Guide](implementation_guide.md) - Implementation details

**Key Sections:**
- Scalability considerations
- Performance optimization
- Alternative approaches
- Future roadmap

### 🎨 Game Designer
*"I want to understand the game mechanics and balance"*

**Start Here:**
1. [Game Design Summary](voidmesh_game_design_summary.md) - Game mechanics inspiration
2. [User Guide](user_guide.md) - Resource economics and balancing
3. [API Documentation](api_documentation.md) - Resource types and behaviors

**Key Sections:**
- Resource spawn behaviors
- Player interaction mechanics
- Economic balancing parameters
- Customization options

## 🔍 Quick Reference

### API Endpoints
```
GET  /health                           # Health check
GET  /api/v1/chunks/{x}/{z}/nodes      # Load chunk
POST /api/v1/harvest/start             # Start harvest
PUT  /api/v1/harvest/sessions/{id}     # Harvest resources
GET  /api/v1/players/{id}/sessions     # Player sessions
```

### Resource Types
- **Iron Ore** (1): Basic mining resource, 100-500 yield, 5-10/hour regen
- **Gold Ore** (2): Valuable resource, 50-300 yield, 2-5/hour regen  
- **Wood** (3): Renewable resource, 50-100 yield, 1/hour regen
- **Stone** (4): Construction material, 75-150 yield, 2/hour regen

### Spawn Types
- **Random Spawn** (0): Appears randomly, respawns elsewhere
- **Static Daily** (1): Fixed location, resets every 24 hours
- **Static Permanent** (2): Always exists, regenerates continuously

### Quality Tiers
- **Poor** (0): Lower yield resources
- **Normal** (1): Standard yield resources  
- **Rich** (2): Higher yield resources

## 📋 Development Workflows

### Quick Setup
```bash
git clone https://github.com/VoidMesh/api.git
cd api && go mod tidy
sqlite3 game.db < internal/db/migrations/001_initial.up.sql
go run ./cmd/server
```

### Documentation Updates
1. Edit relevant markdown files in `.claude/project/docs/`
2. Update this index if adding new documents
3. Test all code examples
4. Submit pull request

### Adding New Features
1. **Design**: Update design docs first
2. **Implement**: Follow [Developer Guide](developer_guide.md)
3. **Document**: Update API docs and user guide
4. **Test**: Add integration tests

## 🤝 Contributing to Documentation

### Documentation Standards

1. **Clear Structure**: Use consistent headings and formatting
2. **Code Examples**: Include working code samples in multiple languages
3. **Error Cases**: Document error conditions and responses
4. **Real Examples**: Use realistic data in examples
5. **Cross-References**: Link between related documents

### Writing Guidelines

- **Audience-First**: Write for the intended audience
- **Practical Focus**: Include actionable information
- **Visual Elements**: Use diagrams, tables, and code blocks
- **Regular Updates**: Keep documentation current with code changes

### Documentation Workflow

1. **Plan**: Identify documentation needs
2. **Draft**: Write initial version with examples
3. **Review**: Test all code examples
4. **Publish**: Update and deploy
5. **Maintain**: Regular updates and improvements

## 🔗 External Resources

### Go Development
- [Go Documentation](https://golang.org/doc/)
- [SQLC Documentation](https://docs.sqlc.dev/)
- [Chi Router](https://go-chi.io/)

### Database
- [SQLite Documentation](https://sqlite.org/docs.html)
- [Database Migration Best Practices](https://github.com/golang-migrate/migrate)

### Game Development
- [EVE Online Development Blog](https://www.eveonline.com/news)
- [Guild Wars 2 API](https://wiki.guildwars2.com/wiki/API)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/VoidMesh/api/issues)
- **Discussions**: [GitHub Discussions](https://github.com/VoidMesh/api/discussions)
- **Documentation Feedback**: Open an issue with the `documentation` label

---

*Last Updated: 2024-01-15*
*Documentation Version: 1.0*