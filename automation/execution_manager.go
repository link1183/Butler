package automation

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type ExecutionManager struct {
	mu         sync.Mutex
	executions map[string]*Execution
	perRule    map[int]string
	rootCtx    context.Context
	logger     *slog.Logger
}

func NewExecutionManager(root context.Context, logger *slog.Logger) *ExecutionManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &ExecutionManager{
		executions: make(map[string]*Execution),
		perRule:    make(map[int]string),
		rootCtx:    root,
		logger:     logger,
	}
}

func (m *ExecutionManager) Start() *Execution {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	ctx, cancel := context.WithCancel(m.rootCtx)

	exec := &Execution{
		ID:     id,
		Ctx:    ctx,
		Cancel: cancel,
	}

	m.executions[id] = exec
	return exec
}

func (m *ExecutionManager) StartWithPolicy(ruleIndex int, mode string) *Execution {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentID, exists := m.perRule[ruleIndex]

	switch mode {

	case "single":
		if exists {
			m.logger.Info("execution skipped",
				slog.Int("rule", ruleIndex),
				slog.String("mode", mode),
				slog.String("reason", "already running"),
			)
			return nil
		}

	case "ignore":
		if exists {
			m.logger.Info("execution ignored",
				slog.Int("rule", ruleIndex),
				slog.String("mode", mode),
				slog.String("reason", "already running"),
			)
			return nil
		}

	case "replace":
		if exists {
			if exec, ok := m.executions[currentID]; ok {
				m.logger.Info("replacing active execution",
					slog.Int("rule", ruleIndex),
					slog.String("previous_execution", currentID),
				)
				exec.Cancel()
				delete(m.executions, currentID)
			}
		}

	case "parallel":
		// allowed

	default:
		m.logger.Warn("unknown execution mode, defaulting to parallel",
			slog.Int("rule", ruleIndex),
			slog.String("mode", mode),
		)
	}

	id := uuid.NewString()
	ctx, cancel := context.WithCancel(m.rootCtx)

	exec := &Execution{
		ID:     id,
		Ctx:    ctx,
		Cancel: cancel,
	}

	m.executions[id] = exec
	m.perRule[ruleIndex] = id

	m.logger.Info("execution started",
		slog.Int("rule", ruleIndex),
		slog.String("execution", id),
		slog.Int("active", len(m.executions)),
	)

	return exec
}

func (m *ExecutionManager) Finish(ruleIndex int, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.executions, id)

	if m.perRule[ruleIndex] == id {
		delete(m.perRule, ruleIndex)
	}

	m.logger.Info("execution finished",
		slog.Int("rule", ruleIndex),
		slog.String("execution", id),
		slog.Int("active", len(m.executions)),
	)
}

func (m *ExecutionManager) Cancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if exec, ok := m.executions[id]; ok {
		exec.Cancel()
		delete(m.executions, id)
	}
}
