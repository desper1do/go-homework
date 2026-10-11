package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"task-tracker/internal/task"
)

// FileStore хранит задачи в памяти и сохраняет их в JSON файл при каждом изменении
type FileStore struct {
	filePath string
	tasks    map[task.TaskID]task.Task
	nextID   task.TaskID
}

// taskFile описывает формат JSON файла
// last_id хранится отдельно от задач, чтобы после удаления задачи с самым большим
// идентификатором ее номер не был выдан повторно новой задаче
type taskFile struct {
	LastID task.TaskID               `json:"last_id"`
	Tasks  map[task.TaskID]task.Task `json:"tasks"`
}

// NewFileStore создает хранилище и загружает задачи из файла
// Если файла еще нет, хранилище начинается пустым, а файл появится при первом сохранении
func NewFileStore(filePath string) (*FileStore, error) {
	t := &FileStore{
		filePath: filePath,
		tasks:    make(map[task.TaskID]task.Task),
		nextID:   1,
	}

	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var f taskFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decode file: %w", err)
	}
	if f.Tasks != nil {
		t.tasks = f.Tasks
	}
	// В файле лежит последний выданный идентификатор, поэтому следующий на единицу больше
	t.nextID = f.LastID + 1

	return t, nil
}

// flush записывает все задачи и последний выданный идентификатор в файл
func (s *FileStore) flush() error {
	f := taskFile{LastID: s.nextID - 1, Tasks: s.tasks}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode file: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func (s *FileStore) Get(id task.TaskID) (task.Task, error) {
	t, exists := s.tasks[id]
	if !exists {
		return task.Task{}, fmt.Errorf("get task %d: %w", id, task.ErrTaskNotFound)
	}
	return t, nil
}

func (s *FileStore) List() ([]task.Task, error) {
	tasks := make([]task.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *FileStore) NextID() task.TaskID {
	id := s.nextID
	s.nextID++
	return id
}

func (s *FileStore) Save(t task.Task) error {
	s.tasks[t.ID] = t
	if err := s.flush(); err != nil {
		return fmt.Errorf("save task %d: %w", t.ID, err)
	}
	return nil
}

func (s *FileStore) Delete(id task.TaskID) error {
	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("delete task %d: %w", id, task.ErrTaskNotFound)
	}
	delete(s.tasks, id)
	if err := s.flush(); err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	return nil
}
