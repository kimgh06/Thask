# Scanner Plugins

Thask supports external scanner plugins for languages beyond Go. A plugin is any executable that analyzes a project directory and outputs dependency information as JSON.

## Plugin Contract

A scanner plugin must:

1. Accept `--path <dir>` as a required argument (the project directory to scan)
2. Optionally accept `--max-files <n>` to limit the number of files processed
3. Write valid JSON to stdout matching the `ImportGraphRequest` format
4. Write warnings/errors to stderr
5. Exit with code 0 on success, non-zero on failure

## Output Format

The plugin must output JSON matching this schema:

```json
{
  "mode": "merge",
  "nodes": [
    {
      "type": "TASK",
      "title": "package/name",
      "status": "IN_PROGRESS",
      "positionX": 0,
      "positionY": 0
    }
  ],
  "edges": [
    {
      "sourceTitle": "package/a",
      "targetTitle": "package/b",
      "edgeType": "depends_on"
    }
  ]
}
```

### Fields

**Node fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `type` | Yes | Node type: `FLOW`, `BRANCH`, `TASK`, `BUG`, `API`, `UI`, `GROUP` |
| `title` | Yes | Display name (typically the package/module path) |
| `status` | No | Default: `IN_PROGRESS` |
| `positionX` | No | Default: 0 (auto-layout will reposition) |
| `positionY` | No | Default: 0 |

**Edge fields:**
| Field | Required | Description |
|-------|----------|-------------|
| `sourceTitle` | Yes | Title of the source node (must match a node title) |
| `targetTitle` | Yes | Title of the target node (must match a node title) |
| `edgeType` | Yes | Edge type: `depends_on`, `blocks`, `related`, `parent_child`, `triggers` |

## Usage

```bash
# Use built-in Go scanner (default)
thask scan --path .

# Use an external plugin
thask scan --plugin ./my-python-scanner --path .

# Dry run with plugin
thask scan --plugin ./my-scanner --path . --dry-run
```

## Example: Minimal Python Plugin

```python
#!/usr/bin/env python3
"""Minimal Thask scanner plugin for Python projects."""
import argparse
import json
import os
import ast
import sys

def scan(path):
    nodes = []
    edges = []
    seen = set()

    for root, dirs, files in os.walk(path):
        # Skip hidden dirs and common non-source dirs
        dirs[:] = [d for d in dirs if not d.startswith('.') and d not in ('venv', 'node_modules', '__pycache__')]

        for f in files:
            if not f.endswith('.py'):
                continue
            filepath = os.path.join(root, f)
            relpath = os.path.relpath(filepath, path)
            module = relpath.replace('/', '.').replace('.py', '')

            if module not in seen:
                seen.add(module)
                nodes.append({
                    "type": "TASK",
                    "title": module,
                    "status": "IN_PROGRESS",
                    "positionX": 0,
                    "positionY": 0,
                })

            try:
                with open(filepath) as fh:
                    tree = ast.parse(fh.read())
                for node in ast.walk(tree):
                    if isinstance(node, ast.ImportFrom) and node.module:
                        if node.module in seen:
                            edges.append({
                                "sourceTitle": module,
                                "targetTitle": node.module,
                                "edgeType": "depends_on",
                            })
            except Exception as e:
                print(f"warning: {filepath}: {e}", file=sys.stderr)

    return {"mode": "merge", "nodes": nodes, "edges": edges}

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--path', required=True)
    parser.add_argument('--max-files', type=int, default=500)
    args = parser.parse_args()

    result = scan(args.path)
    json.dump(result, sys.stdout, indent=2)
```

Save as `my-python-scanner.py`, make executable (`chmod +x`), then:

```bash
thask scan --plugin ./my-python-scanner.py --path ./my-python-project --dry-run
```

## Built-in Scanners

| Language | Built-in | Plugin needed |
|----------|----------|---------------|
| Go | Yes (default) | No |
| Python | No | Yes (example above) |
| TypeScript | No | Community |
| Rust | No | Community |
| Java | No | Community |

## Writing Your Own Plugin

1. Create an executable that accepts `--path <dir>`
2. Parse the project at that path
3. Output JSON to stdout in the format above
4. Test with `thask scan --plugin ./your-plugin --path . --dry-run`
5. Import: `thask scan --plugin ./your-plugin --path . --project <id>`
