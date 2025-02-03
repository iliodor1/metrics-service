package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Storage интерфейс для хранения метрик
type Storage interface {
	UpdateGauge(name string, value float64) error
	UpdateCounter(name string, delta int64) error
}

// MemStorage структура для хранения метрик в памяти
type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage создаёт новое хранилище метрик
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge обновляет или добавляет метрику типа gauge
func (m *MemStorage) UpdateGauge(name string, value float64) error {
	m.gauges[name] = value
	return nil
}

// UpdateCounter обновляет или добавляет метрику типа counter
func (m *MemStorage) UpdateCounter(name string, delta int64) error {
	m.counters[name] += delta
	return nil
}

// Handler структура для хранения зависимостей обработчика
type Handler struct {
	storage Storage
}

// NewHandler создаёт новый экземпляр обработчика
func NewHandler(storage Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

// webhook обработчик для приёма метрик
func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	// Проверка метода запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён. Используйте POST.", http.StatusMethodNotAllowed)
		return
	}

	// Разбор URL
	// Ожидаемый формат: /update/<type>/<name>/<value>
	path := strings.TrimPrefix(r.URL.Path, "/update/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 {
		http.Error(w, "Неверный формат URL. Ожидается /update/<type>/<name>/<value>.", http.StatusBadRequest)
		return
	}

	metricType, metricName, metricValue := parts[0], parts[1], parts[2]

	// Проверка наличия имени метрики
	if metricName == "" {
		http.Error(w, "Имя метрики не может быть пустым.", http.StatusNotFound)
		return
	}

	// Обработка в зависимости от типа метрики
	switch metricType {
	case "gauge":
		// Парсинг значения как float64
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Неверное значение для gauge. Ожидается float64.", http.StatusBadRequest)
			return
		}
		// Обновление метрики
		if err := h.storage.UpdateGauge(metricName, value); err != nil {
			http.Error(w, "Ошибка при обновлении gauge метрики.", http.StatusInternalServerError)
			return
		}
	case "counter":
		// Парсинг значения как int64
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Неверное значение для counter. Ожидается int64.", http.StatusBadRequest)
			return
		}
		// Обновление метрики
		if err := h.storage.UpdateCounter(metricName, delta); err != nil {

			http.Error(w, "Ошибка при обновлении counter метрики.", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Неподдерживаемый тип метрики. Допустимые типы: gauge, counter.", http.StatusBadRequest)
		return
	}

	// Успешный ответ
	w.WriteHeader(http.StatusOK)
}

func main() {
	// Создаём новое хранилище
	storage := NewMemStorage()

	// Создаём новый обработчик с зависимостями
	handler := NewHandler(storage)

	// Регистрируем обработчик для пути /update/
	// Функция ServeMux автоматически передаст запросы, начинающиеся с /update/, этому обработчику
	http.HandleFunc("/update/", handler.webhook)

	// Настройка адреса сервера
	addr := "localhost:8080"
	log.Printf("Сервер запущен на http://%s\n", addr)

	// Запуск HTTP-сервера
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}
}
