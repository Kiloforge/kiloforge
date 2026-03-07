# Implementation Plan: Extend Gitea Client with Issue, Label, and Project Board APIs

**Track ID:** impl-gitea-issue-api_20260308170000Z

## Phase 1: Issue & Label API Methods (4 tasks)

### Task 1.1: Implement CreateIssue
- [x] `CreateIssue(ctx, repo, title, body string, labels []string) → (int, error)`
- [x] POST /api/v1/repos/{owner}/{repo}/issues with title, body, labels
- [x] Parse response for issue number
- [x] Return created issue number

### Task 1.2: Implement UpdateIssue and CloseIssue
- [x] `UpdateIssue(ctx, repo string, issueNum int, title, body, state string) → error`
- [x] PATCH /api/v1/repos/{owner}/{repo}/issues/{number}
- [x] Support updating title, body, and state (open/closed)
- [x] Only send non-empty fields

### Task 1.3: Implement GetIssues
- [x] `GetIssues(ctx, repo, state string, labels []string) → ([]Issue, error)`
- [x] GET /api/v1/repos/{owner}/{repo}/issues with query params
- [x] Define `Issue` struct with Number, Title, State, Labels fields
- [x] Parse JSON array response

### Task 1.4: Implement EnsureLabels
- [x] `EnsureLabels(ctx, repo string, labels []LabelDef) → error`
- [x] Define `LabelDef` struct with Name, Color
- [x] GET existing labels from /api/v1/repos/{owner}/{repo}/labels
- [x] POST only missing labels (diff against existing)
- [x] Idempotent — safe to re-run

## Phase 2: Project Board API Methods (3 tasks)

### Task 2.1: Implement CreateProject and GetProjects
- [x] `CreateProject(ctx, repo, title, description string) → (int, error)`
- [x] POST /api/v1/repos/{owner}/{repo}/projects
- [x] `GetProjects(ctx, repo string) → ([]Project, error)`
- [x] Define `Project` struct with ID, Title

### Task 2.2: Implement CreateColumn and GetColumns
- [x] `CreateColumn(ctx context.Context, projectID int, title string) → (int, error)`
- [x] POST /api/v1/repos/{owner}/{repo}/projects/{id}/columns
- [x] `GetColumns(ctx context.Context, projectID int) → ([]Column, error)`
- [x] Define `Column` struct with ID, Title

### Task 2.3: Implement CreateCard and MoveCard
- [x] `CreateCard(ctx context.Context, columnID, issueID int) → (int, error)`
- [x] POST to column cards endpoint with issue_id
- [x] `MoveCard(ctx context.Context, cardID, columnID int) → error`
- [x] PATCH card to move to target column

## Phase 3: Tests and Port Interface (3 tasks)

### Task 3.1: Tests for issue and label methods
- [x] Table-driven httptest tests for CreateIssue (success, error)
- [x] Tests for UpdateIssue (update fields, close issue)
- [x] Tests for GetIssues (empty, multiple, with filters)
- [x] Tests for EnsureLabels (all new, some existing, all existing)

### Task 3.2: Tests for project board methods
- [x] Tests for CreateProject and GetProjects
- [x] Tests for CreateColumn and GetColumns
- [x] Tests for CreateCard and MoveCard

### Task 3.3: Update port interface
- [x] Add new methods to `port.GiteaClient` interface
- [x] Verify all existing implementations still satisfy interface
- [x] `go build ./...` and `go test -race ./...`

---

**Total: 10 tasks across 3 phases**
