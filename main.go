package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User — структура для парсинга JSON-запроса
type User struct {
	Username string `json:"username"` // имя пользователя
	Email    string `json:"email"`    // email пользователя
}

// Строка подключения к Postgres
const dbURI = "postgres://demo:demo@mypostgres:5432/postgres"

// Глобальная переменная для пула соединений с базой данных
var db *pgxpool.Pool

func main() {
	ctx := context.Background()

	// Инициализируем пул соединений к базе данных Postgres
	var err error
	db, err = pgxpool.New(ctx, dbURI)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	// Закрываем пул соединений при завершении работы приложения
	defer db.Close()

	// Выставляем таймаут для проверки подключения к базе
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Проверяем, что соединение с базой установлено
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}

	// Регистрируем обработчик HTTP-запросов на эндпоинте /users
	http.HandleFunc("/users", createUserHandler)

	fmt.Println("Сервер запущен на :8080")
	// Запускаем HTTP-сервер на порту 8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}

// createUserHandler — обработчик POST-запросов для создания нового пользователя
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что метод запроса — POST
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var user User
	// Парсим JSON-тело запроса в структуру User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	// Проверяем, что оба обязательных поля присутствуют
	if user.Username == "" || user.Email == "" {
		http.Error(w, "Поля username и email обязательны", http.StatusBadRequest)
		return
	}

	// Контекст с таймаутом для выполнения запроса к базе
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// SQL-запрос на вставку данных в таблицу users
	query := `INSERT INTO users (username, email) VALUES ($1, $2)`
	_, err := db.Exec(ctx, query, user.Username, user.Email)
	if err != nil {
		log.Printf("Ошибка вставки в базу: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ клиенту
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Пользователь успешно создан")
}
