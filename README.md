# witr (why-is-this-running)

## 0. Disclaimer
This project is a work in progress. The requirements, scope, and design decisions documented here are subject to change based on implementation feasibility, technical constraints, and learnings as development progresses.

## 1. Purpose

**witr** exists to answer a single question:

> **Why is this running?**

When something is running on a system—whether it is a process, a service, or something bound to a port—there is always a cause. That cause is often indirect, non-obvious, or spread across multiple layers such as supervisors, containers, services, or shells.

Existing tools (`ps`, `top`, `lsof`, `ss`, `systemctl`, `docker ps`) expose state and metadata. They show *what* is running, but leave the user to infer *why* by manually correlating outputs across tools.

**witr** makes that causality explicit.

It explains **where a running thing came from**, **how it was started**, and **what chain of systems is responsible for it existing right now**, in a single, human-readable output.

---

## 2. Goals

### Primary goals
- Explain **why a process exists**, not just that it exists
- Reduce time‑to‑understanding during debugging and outages
- Work with zero configuration
- Be safe, read‑only, and non‑destructive
- Prefer clarity over completeness

### Non‑goals
- Not a monitoring tool
- Not a performance profiler
- Not a replacement for systemd/docker tooling
- Not a remediation or auto‑fix tool

---

## 3. Core Concept

witr treats **everything as a process question**.

Ports, services, containers, and commands all eventually map to **PIDs**. Once a PID is identified, witr builds a causal chain explaining *why that PID exists*.

At its core, witr answers:

1. What is running?
2. How did it start?
3. What is keeping it running?
4. What context does it belong to?

---

## 4. Supported Targets

witr supports multiple entry points that converge to PID analysis.

### 4.1 Name (process or service)
```bash
witr node
witr nginx
```

Accepts a single positional argument that may refer to a process name or a service name. witr intentionally does not require the user to distinguish between the two.

If the name matches **both** a running process and a service definition, witr treats this as ambiguity and asks the user to re-run the command with an explicit PID.

If multiple matching processes exist, witr lists them and requires disambiguation.

---

### 4.2 PID
```bash
witr --pid 14233
```

Explains why a specific process exists.

---

### 4.3 Port
```bash
witr --port 5000
```

Explains the process(es) listening on a port.

---

## 5. Output Behavior


### 5.1 Output principles
- Single screen by default (best effort)
- Deterministic ordering
- Narrative-style explanation
- Best-effort detection with explicit uncertainty

---

### 5.2 Standard Output Sections

#### Target
What the user asked about.

#### Process
Executable, PID, command, and start time.

#### Why It Exists
A causal ancestry chain showing how the process came to exist.

Format:
```
systemd → docker → node → app
```

This is the core value of witr.

#### Source
The primary system responsible for starting or supervising the process.

Examples:
- systemd unit
- docker container
- pm2
- cron
- interactive shell

Only **one primary source** is selected.

#### Context (best effort)
- Working directory
- Git repository name and branch
- Docker container name / image
- Public vs private bind

#### Notes / Warnings
Non‑blocking observations such as:
- Running as root
- Bound to public interface (0.0.0.0 / ::)
- No supervisor detected
- Restarted multiple times

---

## 6. Flags

```text
--pid <n>         Explain a specific PID
--port <n>        Explain port usage

--short           One-line summary
--tree            Show full process ancestry tree
```

A single positional argument (without flags) is treated as a process or service name.

No configuration files. No environment variables.

---

## 7. Example Outputs

### 7.1 Name-based query

```bash
witr node
```

```
Target      : node

Process     : node (pid 14233)
Command     : node index.js
Started     : 2 days ago (Mon 2025-02-02 11:42:10 +0530)

Why It Exists :
  systemd → pm2 → node

Source      : pm2 startup

Working Dir : /opt/apps/expense-manager
Git Repo    : expense-manager (main)
```

---

### 7.2 Port-based query

```bash
witr --port 5000
```

```
Target      : Port 5000

Process     : python (pid 24891)
Command     : python app.py
Started     : 6 hours ago (Tue 01:12 IST, 2025-02-04 01:12:03 +0530)

Why It Exists :
  systemd → docker → python

Source      : docker container
Container   : api-dev
Image       : python:3.11-slim

Working Dir : /app
Git Repo    : expense-manager (branch: dev)

Listening   : 0.0.0.0:5000 (public)

Notes       :
  • Bound to public interface (0.0.0.0)
  • Restarted 3 times
  • No healthcheck detected
```

---

### 7.3 Multiple matches

#### 7.3.1 Multiple matching processes

```bash
witr node
```

```
Found 3 matching running entities:

[1] PID 12091  node server.js  (docker)
[2] PID 14233  node index.js   (pm2)
[3] PID 18801  node worker.js  (manual)

Re-run with:
  witr --pid <pid>
```

---

#### 7.3.2 Ambiguous name (process and service)

```bash
witr nginx
```

```
Ambiguous target: "nginx"

The name matches multiple entities:

[1] PID 2311   nginx: master process   (service)
[2] PID 24891  nginx: worker process   (manual)

witr cannot determine intent safely.
Please re-run with an explicit PID:
  witr --pid <pid>
```

---

### 7.4 Short output

```bash
witr --port 5000 --short
```

```
Port 5000 → python (pid 24891) | systemd → docker → python | started Tue 01:12 IST, 2025-02-04
```

---

### 7.5 Tree output

```bash
witr --pid 14233 --tree
```

```
systemd (pid 1)
└─ pm2 (pid 812)
   └─ node (pid 913)
      └─ node index.js (pid 14233)
```

---

## 8. Platform Support

- Linux

---

## 9. Safety & Trust

- Read‑only by default
- No automatic killing or modification of processes
- Explicit warnings instead of implicit actions
- Never escalates privileges silently

---

## 10. Success Criteria

witr is successful if:
- An engineer can answer "why is this running?" within seconds
- It reduces reliance on multiple tools
- Output is understandable under stress
- Users trust it during incidents

---

