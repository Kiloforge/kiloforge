# Implementation Plan: Research: Global Gitea Server for Multi-Project Coordination

**Track ID:** research-global-gitea-multiproject_20260307120001Z

## Phase 1: Architecture Research

### Task 1.1: Design global vs per-project separation
- Define what is "global" (Gitea instance, relay server, ports) vs "per-project" (repo, webhooks, agents, state)
- Propose data directory structure: `~/.crelay/` global vs per-project
- Consider: should the relay be global too, or per-project?
- **Output:** Architecture diagram / description

### Task 1.2: Design config schema evolution
- Current: flat `config.json` with single project fields
- Proposed: global config + per-project registration
- Define both schemas with examples
- Plan migration from old to new format
- **Output:** Config schema proposal

### Task 1.3: Design state model evolution
- Current: flat `agents` array with no project association
- Proposed: agents tagged with project ID, or separate state per project
- Consider querying: "show all agents" vs "show agents for this project"
- **Output:** State schema proposal

### Verification 1
- [ ] Global vs per-project boundary defined
- [ ] Config schema proposed
- [ ] State model proposed

## Phase 2: Workflow & Integration Research

### Task 2.1: Design project onboarding flow
- What happens when user runs `crelay add` in a project directory?
- Steps: create repo in Gitea, add git remote, create webhook, register in config
- Handle: project already registered, project dir moved, repo name conflicts
- **Output:** Onboarding flow document

### Task 2.2: Design webhook routing strategy
- Gitea webhook payload includes `repository.full_name` — can be used for routing
- Option A: Single relay, routes by repo name to correct project context
- Option B: Separate webhook URLs per project (e.g., `/webhook/{project}`)
- Evaluate: simplicity, reliability, debuggability
- **Output:** Routing strategy recommendation

### Task 2.3: Design CLI command restructuring
- Current: init, status, agents, logs, attach, stop, destroy
- Proposed additions: add, remove, list (projects)
- Proposed changes: init becomes global-only, destroy gains project scope
- All commands need project context (auto-detect from cwd, or explicit flag)
- **Output:** CLI command tree proposal

### Verification 2
- [ ] Onboarding flow documented
- [ ] Webhook routing strategy chosen
- [ ] CLI restructuring proposed

## Phase 3: Migration & Final Report

### Task 3.1: Design migration path
- How does existing single-project crelay upgrade to multi-project?
- Auto-detect old config format and migrate
- Preserve existing Gitea data and agent state
- **Output:** Migration strategy

### Task 3.2: Write final research document
- Compile all findings into `docs/research-global-gitea-multiproject.md`
- Include: architecture, schemas, flows, CLI changes, migration, open questions
- Highlight decisions that need user input
- **Output:** Complete research document

### Verification 3
- [ ] Migration path documented
- [ ] Research document written and committed
