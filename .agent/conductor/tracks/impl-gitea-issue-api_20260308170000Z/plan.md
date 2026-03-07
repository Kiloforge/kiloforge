# Implementation Plan: Extend Gitea Client with Issue, Label, and Project Board APIs

**Track ID:** impl-gitea-issue-api_20260308170000Z

## Phase 1: Issue & Label API Methods (4 tasks)

### Task 1.1: Implement CreateIssue
- [ ] `CreateIssue(ctx, repo, title, body string, labels []string) → (int, error)`
- [ ] POST /api/v1/repos/{owner}/{repo}/issues with title, body, labels
- [ ] Parse response for issue number
- [ ] Return created issue number

### Task 1.2: Implement UpdateIssue and CloseIssue
- [ ] `UpdateIssue(ctx, repo string, issueNum int, title, body, state string) → error`
- [ ] PATCH /api/v1/repos/{owner}/{repo}/issues/{number}
- [ ] Support updating title, body, and state (open/closed)
- [ ] Only send non-empty fields

### Task 1.3: Implement GetIssues
- [ ] `GetIssues(ctx, repo, state string, labels []string) → ([]Issue, error)`
- [ ] GET /api/v1/repos/{owner}/{repo}/issues with query params
- [ ] Define `Issue` struct with Number, Title, State, Labels fields
- [ ] Parse JSON array response

### Task 1.4: Implement EnsureLabels
- [ ] `EnsureLabels(ctx, repo string, labels []LabelDef) → error`
- [ ] Define `LabelDef` struct with Name, Color
- [ ] GET existing labels from /api/v1/repos/{owner}/{repo}/labels
- [ ] POST only missing labels (diff against existing)
- [ ] Idempotent — safe to re-run

## Phase 2: Project Board API Methods (3 tasks)

### Task 2.1: Implement CreateProject and GetProjects
- [ ] `CreateProject(ctx, repo, title, description string) → (int, error)`
- [ ] POST /api/v1/repos/{owner}/{repo}/projects
- [ ] `GetProjects(ctx, repo string) → ([]Project, error)`
- [ ] Define `Project` struct with ID, Title

### Task 2.2: Implement CreateColumn and GetColumns
- [ ] `CreateColumn(ctx context.Context, projectID int, title string) → (int, error)`
- [ ] POST /api/v1/repos/{owner}/{repo}/projects/{id}/columns
- [ ] `GetColumns(ctx context.Context, projectID int) → ([]Column, error)`
- [ ] Define `Column` struct with ID, Title

### Task 2.3: Implement CreateCard and MoveCard
- [ ] `CreateCard(ctx context.Context, columnID, issueID int) → (int, error)`
- [ ] POST to column cards endpoint with issue_id
- [ ] `MoveCard(ctx context.Context, cardID, columnID int) → error`
- [ ] PATCH card to move to target column

## Phase 3: Tests and Port Interface (3 tasks)

### Task 3.1: Tests for issue and label methods
- [ ] Table-driven httptest tests for CreateIssue (success, error)
- [ ] Tests for UpdateIssue (update fields, close issue)
- [ ] Tests for GetIssues (empty, multiple, with filters)
- [ ] Tests for EnsureLabels (all new, some existing, all existing)

### Task 3.2: Tests for project board methods
- [ ] Tests for CreateProject and GetProjects
- [ ] Tests for CreateColumn and GetColumns
- [ ] Tests for CreateCard and MoveCard

### Task 3.3: Update port interface
- [ ] Add new methods to `port.GiteaClient` interface
- [ ] Verify all existing implementations still satisfy interface
- [ ] `go build ./...` and `go test -race ./...`

---

**Total: 10 tasks across 3 phases**
