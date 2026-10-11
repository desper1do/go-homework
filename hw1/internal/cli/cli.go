package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"task-tracker/internal/service"
	"task-tracker/internal/task"
	"time"
)

// TaskCLI принимает команды пользователя из стандартного ввода и передает их сервису
type TaskCLI struct {
	svc *service.TaskService
}

// NewTaskCLI создает интерфейс командной строки поверх переданного сервиса
func NewTaskCLI(svc *service.TaskService) *TaskCLI {
	return &TaskCLI{svc: svc}
}

// Run читает команды построчно и выполняет их до команды exit или конца ввода
func (c *TaskCLI) Run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "exit":
			return
		case "help":
			c.printHelp()
		case "add":
			c.handleAdd(parts[1:])
		case "list":
			c.handleList()
		case "get":
			c.handleGet(parts[1:])
		case "update":
			c.handleUpdate(parts[1:])
		case "delete":
			c.handleDelete(parts[1:])
		case "status":
			c.handleStatus(parts[1:])
		default:
			fmt.Printf("Неизвестная команда: %q. Введите 'help' для списка доступных команд.\n", parts[0])
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Ошибка при чтении ввода: %v\n", err)
	}
}

func (c *TaskCLI) printHelp() {
	fmt.Println("add <текст> <дата ГГГГ-ММ-ДД>                              - добавить новую задачу")
	fmt.Println("get <идентификатор>                                        - показать задачу по идентификатору")
	fmt.Println("update <идентификатор> <новый заголовок> <новая дата>      - обновить задачу")
	fmt.Println("delete <идентификатор>                                     - удалить задачу")
	fmt.Println("status <идентификатор> <planned|in_progress|done|canceled> - изменить статус задачи")
	fmt.Println("list                                                       - показать список задач")
	fmt.Println("help                                                       - показать список доступных команд")
	fmt.Println("exit                                                       - выйти из приложения")
}

func (c *TaskCLI) handleAdd(args []string) {
	if len(args) < 2 {
		fmt.Println("Недостаточно аргументов для команды 'add'. Использование: add <текст> <дата ГГГГ-ММ-ДД>")
		return
	}

	titleStr := strings.Join(args[:len(args)-1], " ")
	date := args[len(args)-1]

	title, err := parseTitle(titleStr)
	if err != nil {
		fmt.Printf("Ошибка при разборе заголовка: %v\n", err)
		return
	}

	dateTime, err := parseDate(date)
	if err != nil {
		fmt.Printf("Ошибка при разборе даты: %v\n", err)
		return
	}

	createdTask, err := c.svc.Add(title, dateTime)
	switch {
	case errors.Is(err, service.ErrDeadlineInPast):
		fmt.Printf("Невозможно добавить задачу с прошедшим сроком выполнения\n")
		return
	case errors.Is(err, service.ErrEmptyTitle):
		fmt.Printf("Заголовок задачи не может быть пустым\n")
		return
	case err != nil:
		fmt.Printf("Ошибка при добавлении задачи: %v\n", err)
		return
	default:
		c.printTask(createdTask)
	}
}

func parseDate(s string) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("неверная дата %q, нужен формат ГГГГ-ММ-ДД", s)
	}
	return d, nil
}

func parseTitle(s string) (string, error) {
	title := strings.TrimSpace(s)
	if title == "" {
		return "", fmt.Errorf("заголовок не может быть пустым")
	}
	return title, nil
}

func parseID(s string) (task.TaskID, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("неверный идентификатор задачи %q", s)
	}
	return task.TaskID(n), nil
}

func (c *TaskCLI) handleList() {
	tasks, err := c.svc.List()
	if err != nil {
		fmt.Printf("Ошибка при получении списка задач: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		fmt.Println("Список задач пуст.")
	} else {
		for _, t := range tasks {
			c.printTask(t)
		}
	}
}

func (c *TaskCLI) handleGet(args []string) {
	if len(args) == 0 {
		fmt.Println("Недостаточно аргументов для команды 'get'. Использование: get <идентификатор>")
		return
	}

	id, err := parseID(args[0])
	if err != nil {
		fmt.Printf("Ошибка при разборе идентификатора: %v\n", err)
		return
	}

	foundTask, err := c.svc.GetOne(id)
	if errors.Is(err, task.ErrTaskNotFound) {
		fmt.Printf("Задача с идентификатором %d не найдена\n", id)
		return
	}
	if err != nil {
		fmt.Printf("Ошибка при получении задачи: %v\n", err)
		return
	}

	c.printTask(foundTask)
}

func (c *TaskCLI) handleDelete(args []string) {
	if len(args) == 0 {
		fmt.Println("Недостаточно аргументов для команды 'delete'. Использование: delete <идентификатор>")
		return
	}

	id, err := parseID(args[0])
	if err != nil {
		fmt.Printf("Ошибка при разборе идентификатора: %v\n", err)
		return
	}

	err = c.svc.Delete(id)
	if errors.Is(err, task.ErrTaskNotFound) {
		fmt.Printf("Задача с идентификатором %d не найдена\n", id)
		return
	}
	if err != nil {
		fmt.Printf("Ошибка при удалении задачи: %v\n", err)
		return
	}

	fmt.Printf("Задача с идентификатором %d успешно удалена\n", id)
}

func (c *TaskCLI) handleStatus(args []string) {
	if len(args) != 2 {
		fmt.Println("Недостаточно аргументов для команды 'status'. Использование: status <идентификатор> <новый статус>")
		return
	}

	id, err := parseID(args[0])
	if err != nil {
		fmt.Printf("Ошибка при разборе идентификатора: %v\n", err)
		return
	}

	newStatus := task.Status(args[1])
	err = c.svc.SetStatus(id, newStatus)

	switch {
	case errors.Is(err, task.ErrTaskNotFound):
		fmt.Printf("Задача с идентификатором %d не найдена\n", id)
		return
	case errors.Is(err, service.ErrInvalidStatus):
		fmt.Printf("Такого статуса нет. Допустимые: %s, %s, %s, %s\n", task.Planned, task.InProgress, task.Done, task.Canceled)
		return
	case errors.Is(err, service.ErrTaskFinished):
		fmt.Printf("Невозможно изменить статус завершенной или отмененной задачи\n")
		return
	case errors.Is(err, service.ErrSameStatus):
		fmt.Printf("У задачи с идентификатором %d статус уже установлен на %s\n", id, newStatus)
		return
	case err != nil:
		fmt.Printf("Ошибка при изменении статуса задачи: %v\n", err)
		return
	default:
		fmt.Printf("Статус задачи с идентификатором %d успешно изменен на %s\n", id, newStatus)
	}
}

func (c *TaskCLI) handleUpdate(args []string) {
	if len(args) < 1 {
		fmt.Println("Недостаточно аргументов для команды 'update'. Использование: update <идентификатор> [<новый заголовок>] [<новый срок выполнения>]")
		return
	}
	var title *string
	var deadline *time.Time

	id, err := parseID(args[0])
	if err != nil {
		fmt.Printf("Ошибка при разборе идентификатора: %v\n", err)
		return
	}

	lastArg := args[len(args)-1]
	if parsedDate, err := parseDate(lastArg); err == nil {
		deadline = &parsedDate
		args = args[:len(args)-1]
	}

	if len(args) > 1 {
		t, err := parseTitle(strings.Join(args[1:], " "))
		if err != nil {
			fmt.Printf("Ошибка при разборе заголовка: %v\n", err)
			return
		}
		title = &t
	}

	if title == nil && deadline == nil {
		fmt.Println("Нужно указать хотя бы один параметр для обновления: новый заголовок или новый срок выполнения.")
		return
	}

	err = c.svc.Update(id, title, deadline)
	switch {
	case errors.Is(err, task.ErrTaskNotFound):
		fmt.Printf("Задача с идентификатором %d не найдена\n", id)
		return
	case errors.Is(err, service.ErrTaskFinished):
		fmt.Printf("Невозможно изменить завершенную или отмененную задачу\n")
		return
	case errors.Is(err, service.ErrDeadlineInPast):
		fmt.Printf("Невозможно установить срок выполнения в прошлом\n")
		return
	case errors.Is(err, service.ErrEmptyTitle):
		fmt.Printf("Заголовок задачи не может быть пустым\n")
		return
	case err != nil:
		fmt.Printf("Ошибка при обновлении задачи: %v\n", err)
		return
	default:
		fmt.Printf("Задача с идентификатором %d успешно обновлена\n", id)
	}
}

func (c *TaskCLI) printTask(t task.Task) {
	fmt.Printf("ID=%d, %s (срок %s). Статус: %s\n", t.ID, t.Title, t.Deadline.Format("2006-01-02"), t.Status)
}
