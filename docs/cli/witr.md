## witr

Explain processes

### Synopsis

witr explains processes and their ancestry, showing how they were started and what they are doing.

```
witr [process name] [flags]
```

### Examples

```

  # Inspect a running process by name
  witr nginx

  # Look up a process by PID
  witr --pid 1234

  # Find the process listening on a specific port
  witr --port 5432

  # Show the full process ancestry (who started whom)
  witr postgres --tree

  # Show only warnings (suspicious env, arguments, parents)
  witr docker --warnings

  # Display only environment variables of the process
  witr node --env

  # Short, single-line output (useful for scripts)
  witr sshd --short

  # Disable colorized output (CI or piping)
  witr redis --no-color

  # Output machine-readable JSON
  witr chrome --json

  # Combine flags: inspect port, show ancestry, output JSON
  witr --port 8080 --tree --json

```

### Options

```
      --env           show only environment variables for the process
  -h, --help          help for witr
      --json          output as JSON
      --no-color      disable colorized output
      --pid string    pid to look up
      --port string   port to look up
      --short         short output
      --tree          tree output
      --warnings      show only warnings
```

