package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

type Database struct {
	conn *sql.DB
}

// InitDB initializes the database connection and creates tables
func InitDB() (*Database, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/mcp-tracker.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{conn: db}
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

func (db *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		agent_id TEXT,
		title TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		priority INTEGER DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		completed_at DATETIME,
		FOREIGN KEY (project_id) REFERENCES projects(id),
		FOREIGN KEY (agent_id) REFERENCES agents(id)
	);

	CREATE TABLE IF NOT EXISTS documentation (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (task_id) REFERENCES tasks(id)
	);

	CREATE INDEX IF NOT EXISTS idx_agents_project ON agents(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_agent ON tasks(agent_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_documentation_task ON documentation(task_id);
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *Database) Close() error {
	return db.conn.Close()
}

// Project operations
func (db *Database) CreateProject(project *models.Project) error {
	query := `INSERT INTO projects (id, name, description, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, project.ID, project.Name, project.Description,
		project.CreatedAt, project.UpdatedAt)
	return err
}

func (db *Database) GetProject(id string) (*models.Project, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?`
	project := &models.Project{}
	err := db.conn.QueryRow(query, id).Scan(&project.ID, &project.Name, &project.Description,
		&project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (db *Database) ListProjects() ([]*models.Project, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM projects ORDER BY created_at DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		project := &models.Project{}
		if err := rows.Scan(&project.ID, &project.Name, &project.Description,
			&project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// Agent operations
func (db *Database) CreateAgent(agent *models.Agent) error {
	query := `INSERT INTO agents (id, project_id, name, role, status, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, agent.ID, agent.ProjectID, agent.Name, agent.Role,
		agent.Status, agent.CreatedAt, agent.UpdatedAt)
	return err
}

func (db *Database) GetAgent(id string) (*models.Agent, error) {
	query := `SELECT id, project_id, name, role, status, created_at, updated_at FROM agents WHERE id = ?`
	agent := &models.Agent{}
	err := db.conn.QueryRow(query, id).Scan(&agent.ID, &agent.ProjectID, &agent.Name,
		&agent.Role, &agent.Status, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (db *Database) ListAgentsByProject(projectID string) ([]*models.Agent, error) {
	query := `SELECT id, project_id, name, role, status, created_at, updated_at 
	          FROM agents WHERE project_id = ? ORDER BY created_at DESC`
	rows, err := db.conn.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*models.Agent
	for rows.Next() {
		agent := &models.Agent{}
		if err := rows.Scan(&agent.ID, &agent.ProjectID, &agent.Name, &agent.Role,
			&agent.Status, &agent.CreatedAt, &agent.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

// Task operations
func (db *Database) CreateTask(task *models.Task) error {
	query := `INSERT INTO tasks (id, project_id, agent_id, title, description, status, priority, 
	          created_at, updated_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, task.ID, task.ProjectID, task.AgentID, task.Title,
		task.Description, task.Status, task.Priority, task.CreatedAt, task.UpdatedAt, task.CompletedAt)
	return err
}

func (db *Database) GetTask(id string) (*models.Task, error) {
	query := `SELECT id, project_id, agent_id, title, description, status, priority, 
	          created_at, updated_at, completed_at FROM tasks WHERE id = ?`
	task := &models.Task{}
	var completedAt sql.NullTime
	err := db.conn.QueryRow(query, id).Scan(&task.ID, &task.ProjectID, &task.AgentID,
		&task.Title, &task.Description, &task.Status, &task.Priority,
		&task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	return task, nil
}

func (db *Database) UpdateTaskStatus(id string, status models.TaskStatus) error {
	now := time.Now()
	var completedAt *time.Time
	if status == models.StatusCompleted {
		completedAt = &now
	}

	query := `UPDATE tasks SET status = ?, updated_at = ?, completed_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, status, now, completedAt, id)
	return err
}

func (db *Database) ListTasksByAgent(agentID string) ([]*models.Task, error) {
	query := `SELECT id, project_id, agent_id, title, description, status, priority, 
	          created_at, updated_at, completed_at FROM tasks WHERE agent_id = ? ORDER BY priority DESC, created_at DESC`
	rows, err := db.conn.Query(query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var completedAt sql.NullTime
		if err := rows.Scan(&task.ID, &task.ProjectID, &task.AgentID, &task.Title,
			&task.Description, &task.Status, &task.Priority, &task.CreatedAt,
			&task.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (db *Database) ListTasksByProject(projectID string) ([]*models.Task, error) {
	query := `SELECT id, project_id, agent_id, title, description, status, priority, 
	          created_at, updated_at, completed_at FROM tasks WHERE project_id = ? ORDER BY priority DESC, created_at DESC`
	rows, err := db.conn.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var completedAt sql.NullTime
		if err := rows.Scan(&task.ID, &task.ProjectID, &task.AgentID, &task.Title,
			&task.Description, &task.Status, &task.Priority, &task.CreatedAt,
			&task.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Documentation operations
func (db *Database) CreateDocumentation(doc *models.Documentation) error {
	query := `INSERT INTO documentation (id, task_id, content, created_by, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, doc.ID, doc.TaskID, doc.Content, doc.CreatedBy,
		doc.CreatedAt, doc.UpdatedAt)
	return err
}

func (db *Database) GetDocumentationByTask(taskID string) ([]*models.Documentation, error) {
	query := `SELECT id, task_id, content, created_by, created_at, updated_at 
	          FROM documentation WHERE task_id = ? ORDER BY created_at DESC`
	rows, err := db.conn.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Documentation
	for rows.Next() {
		doc := &models.Documentation{}
		if err := rows.Scan(&doc.ID, &doc.TaskID, &doc.Content, &doc.CreatedBy,
			&doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
