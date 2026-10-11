package task

import (
	"errors"
	"time"
)

// Task описывает задачу, которая может быть запланирована, выполняться, отменена или завершена
type Task struct {
	ID       TaskID
	Title    string
	Status   Status
	Deadline time.Time
}

// Status представляет статус задачи
type Status string

// Допустимые значения Status
const (
	Planned    Status = "planned"
	InProgress Status = "in_progress"
	Canceled   Status = "canceled"
	Done       Status = "done"
)

// TaskID представляет уникальный идентификатор задачи
type TaskID int

// IsValid проверяет, является ли статус задачи допустимым
func (s Status) IsValid() bool {
	switch s {
	case Planned, InProgress, Canceled, Done:
		return true
	default:
		return false
	}
}

// ErrTaskNotFound возвращается, когда задачи с указанным идентификатором нет в хранилище
var ErrTaskNotFound = errors.New("задача с указанным идентификатором не найдена")
