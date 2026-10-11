package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"task-tracker/internal/task"
	"time"
)

// TaskStore описывает хранилище задач, которое нужно сервису
// Сервис зависит от интерфейса, поэтому хранилище можно заменить без изменения бизнес логики
type TaskStore interface {
	Save(t task.Task) error
	Get(id task.TaskID) (task.Task, error)
	List() ([]task.Task, error)
	Delete(id task.TaskID) error
	NextID() task.TaskID
}

// TaskService содержит бизнес правила работы с задачами и сохраняет их через TaskStore
type TaskService struct {
	store TaskStore
}

// NewTaskService создает сервис, который работает с переданным хранилищем
func NewTaskService(store TaskStore) *TaskService {
	return &TaskService{store: store}
}

// Add создает задачу со статусом Planned
// Срок выполнения сравнивается с началом текущего дня, а не с текущим моментом,
// чтобы можно было назначить задачу на сегодня
func (t *TaskService) Add(title string, deadline time.Time) (task.Task, error) {
	now := time.Now()
	y, m, d := now.Date()
	startOfDay := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	if deadline.Before(startOfDay) {
		return task.Task{}, fmt.Errorf("add task: %w", ErrDeadlineInPast)
	} else if strings.TrimSpace(title) == "" {
		return task.Task{}, fmt.Errorf("add task: %w", ErrEmptyTitle)
	} else {
		newTask := task.Task{
			ID:       t.store.NextID(),
			Title:    title,
			Status:   task.Planned,
			Deadline: deadline,
		}
		if err := t.store.Save(newTask); err != nil {
			return task.Task{}, fmt.Errorf("add task: %w", err)
		}
		return newTask, nil
	}
}

// GetOne возвращает задачу по идентификатору
func (t *TaskService) GetOne(id task.TaskID) (task.Task, error) {
	found, err := t.store.Get(id)
	if err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	return found, nil
}

// Delete удаляет задачу по идентификатору
func (t *TaskService) Delete(id task.TaskID) error {
	if err := t.store.Delete(id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// SetStatus меняет статус задачи
// Завершенную или отмененную задачу изменить нельзя, а установка того же статуса считается ошибкой
func (t *TaskService) SetStatus(id task.TaskID, status task.Status) error {
	taskToUpdate, err := t.store.Get(id)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}

	if !(status.IsValid()) {
		return fmt.Errorf("set status: %w", ErrInvalidStatus)
	}

	if taskToUpdate.Status == task.Done || taskToUpdate.Status == task.Canceled {
		return fmt.Errorf("set status: %w", ErrTaskFinished)
	}
	if taskToUpdate.Status == status {
		return fmt.Errorf("set status: %w", ErrSameStatus)
	}

	taskToUpdate.Status = status

	if err := t.store.Save(taskToUpdate); err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	return nil
}

// Update меняет название и срок выполнения задачи
// Значение nil в title или deadline означает, что это поле менять не нужно
// Завершенную или отмененную задачу изменить нельзя
func (t *TaskService) Update(id task.TaskID, title *string, deadline *time.Time) error {
	taskToUpdate, err := t.store.Get(id)

	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if taskToUpdate.Status == task.Done || taskToUpdate.Status == task.Canceled {
		return fmt.Errorf("update task: %w", ErrTaskFinished)
	}

	now := time.Now()
	y, m, d := now.Date()
	startOfDay := time.Date(y, m, d, 0, 0, 0, 0, now.Location())

	if deadline != nil && deadline.Before(startOfDay) {
		return fmt.Errorf("update task: %w", ErrDeadlineInPast)
	} else if title != nil && strings.TrimSpace(*title) == "" {
		return fmt.Errorf("update task: %w", ErrEmptyTitle)
	}

	if title != nil {
		taskToUpdate.Title = *title
	}

	if deadline != nil {
		taskToUpdate.Deadline = *deadline
	}

	if err := t.store.Save(taskToUpdate); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// List возвращает задачи, отсортированные по сроку выполнения, а при равных сроках по идентификатору
// Сортировка нужна, потому что хранилища отдают задачи из map, а порядок обхода map в Go случайный
func (t *TaskService) List() ([]task.Task, error) {
	tasks, err := t.store.List()
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Deadline.Equal(tasks[j].Deadline) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].Deadline.Before(tasks[j].Deadline)
	})
	return tasks, nil
}

var ErrEmptyTitle = errors.New("название задачи не может быть пустым")
var ErrDeadlineInPast = errors.New("срок выполнения задачи не может быть в прошлом")
var ErrInvalidStatus = errors.New("недопустимый статус задачи")
var ErrSameStatus = errors.New("статус задачи не может быть изменен на тот же самый статус")
var ErrTaskFinished = errors.New("задача завершена или отменена, её нельзя изменять")
