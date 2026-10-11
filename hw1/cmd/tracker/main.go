package main

import (
	"fmt"
	"os"
	"task-tracker/internal/cli"
	"task-tracker/internal/service"
	"task-tracker/internal/storage"
)

func main() {
	taskStore, err := storage.NewFileStore("tasks.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при инициализации хранилища задач: %v\n", err)
		os.Exit(1)
	}

	taskService := service.NewTaskService(taskStore)
	taskCLI := cli.NewTaskCLI(taskService)

	fmt.Println("Добро пожаловать в Task Tracker!")
	fmt.Println("Введите 'help' для списка доступных команд.")
	taskCLI.Run()
	fmt.Println("Выход из Task Tracker. До свидания!")
}
