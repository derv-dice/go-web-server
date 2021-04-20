package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const helloMsgTmpl = `Hello, from service. Today is %s`

// helloHandler - Обработчик метода GET /hello
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("hello handler")
	var err error
	var status int
	var data []byte

	// Этот код выполнится в конце функции
	defer func() {
		// Если перед завершением функции переменная var содержит ошибку, то клиенту вернется текст ошибки
		if err != nil {
			data, err = json.Marshal(response{Error: err.Error()})
			if err != nil {
				status = http.StatusInternalServerError
			}

			w.Write(data)
			return
		}

		// Если перед завершением функции переменная var не содержит ошибку, то клиенту вернутся данные из data
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}()

	// Обрабатываем только метод GET
	switch r.Method {
	case http.MethodGet:
		// Вычисляем текущее время и подставляем его в форматированную строку helloMsgTmpl
		currentTime := time.Now().Format(time.RFC1123Z)

		// Сериализация данных из структуры response в массив байт data
		data, err = json.Marshal(response{Data: fmt.Sprintf(helloMsgTmpl, currentTime)})
		if err != nil {
			status = http.StatusInternalServerError
			return
		}

	default:
		err = fmt.Errorf("метод %q не поддерживается", r.Method)
		status = http.StatusNotImplemented
		return
	}
}

// recovery - Middleware, предотвращающий остановку приложения в случае критической ошибки
func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("panic middleware")

		defer func() {
			fmt.Println("panic middleware defer")
			err := recover()
			if err != nil {
				// В случае непредвиденной критической ошибки - возвращается ответ с формате JSON заданной структуры
				var data []byte
				data, _ = json.Marshal(response{Error: fmt.Sprintf("%v", err)})
				w.WriteHeader(http.StatusInternalServerError) // Важно сначала передать заголовок с статус кодом
				w.Write(data)                                 // А уже после заголовков передается тело ответа

				// Логирование факта ошибки
				log.Printf("panic: {method: %s, ip: %s, url: %s}",
					r.Method,     // HTTP метод
					r.RemoteAddr, // IP адрес отправителя запроса
					r.URL.Path,   // URL метода, на который был отправлен запрос
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// accessLog - Middleware, логирующий все входящие запросы
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("access_log middleware")

		start := time.Now()  // Засекается момент времени, когда непосредственно началась обработка запроса
		next.ServeHTTP(w, r) // Обработка запроса

		log.Printf("access_log: {method: %s, ip: %s, url: %s, time: %s}",
			r.Method,          // HTTP метод
			r.RemoteAddr,      // IP адрес отправителя запроса
			r.URL.Path,        // URL метода, на который был отправлен запрос
			time.Since(start), // Записывается время, прошедшее с момента начала обработки
		)
	})
}

// response - структура, описывающая общий ответ сервера на запросы
type response struct {
	Data  string `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func main() {
	// Создание пустой серверной шины
	mux := http.NewServeMux()

	// регистрация обработчика по адресу /hello
	mux.HandleFunc("/hello", helloHandler)

	// Добавление middleware
	handler := accessLog(mux)
	handler = recovery(handler)

	// запуск сервера по адресу localhost:8080 с собранным обработчиком
	log.Fatal(http.ListenAndServe(":8080", handler))
}