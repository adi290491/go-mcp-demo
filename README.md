# Building MCP Agents in Go

Learning projects for building AI agents in Go that use tools through the
[Model Context Protocol](https://modelcontextprotocol.io) (MCP), with the official
[Go SDK](https://github.com/modelcontextprotocol/go-sdk).

## weather

An MCP server and client covering the basics: the client starts the server as a child
process, talks to it over stdio, lists its tools and calls them. The server's two tools
wrap the US National Weather Service API (`api.weather.gov`):

| Tool | Arguments |
|---|---|
| `get_forecast` | `latitude`, `longitude` |
| `get_alerts` | `state` (two-letter US state code, e.g. `CA`) |

## License

See [LICENSE](LICENSE).
