package storage

import (
	"fmt"
	"task-tracker/internal/task"
)

// MemoryStore хранит задачи только в памяти, после завершения программы данные теряются
type MemoryStore struct {
	tasks  map[task.TaskID]task.Task
	nextID task.TaskID
}

// NewMemoryStore создает пустое хранилище, идентификаторы в нем начинаются с единицы
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks:  make(map[task.TaskID]task.Task),
		nextID: 1,
	}
}

func (m *MemoryStore) Save(t task.Task) error {
	m.tasks[t.ID] = t
	return nil
}

func (m *MemoryStore) Get(id task.TaskID) (task.Task, error) {
	t, exists := m.tasks[id]
	if !exists {
		return task.Task{}, fmt.Errorf("get task %d: %w", id, task.ErrTaskNotFound)
	}
	return t, nil
}

func (m *MemoryStore) List() ([]task.Task, error) {
	tasks := make([]task.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (m *MemoryStore) Delete(id task.TaskID) error {
	if _, exists := m.tasks[id]; !exists {
		return fmt.Errorf("delete task %d: %w", id, task.ErrTaskNotFound)
	}
	delete(m.tasks, id)
	return nil
}

func (m *MemoryStore) NextID() task.TaskID {
	id := m.nextID
	m.nextID++
	return id
}
