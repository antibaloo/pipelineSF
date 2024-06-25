package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antibaloo/ring"
)

// Горутина, принимающая из консоли данные и отправляющая их на первый этап обработки
func ConsoleInput() (<-chan int, <-chan bool) {
	source := make(chan int)
	done := make(chan bool)
	reader := bufio.NewReader(os.Stdin)
	go func() {
		defer close(done)
		for {
			fmt.Println("Введите целое число или команду `exit`, чтобы остановить работу программы: ")
			log.Println("Запись лога: Выведена подсказка пользователю по работе программы.")
			command, _ := reader.ReadString('\n')
			command = strings.Replace(command, "\r", "", -1)
			command = strings.Replace(command, "\n", "", -1)
			switch command {
			case "exit":
				log.Println("Запись лога: Введена команда завершения работы.")
				fmt.Println("Программа завершила работу!")
				return
			default:
				number, err := strconv.Atoi(command)
				if err != nil {
					log.Printf("Запись лога: Введеные данные %s не являются целым числом или командой остановки работы. Выведено сообщение об ошибке\n", command)
					fmt.Println("Программа принимает к обработке только целые числа!")
					continue
				}
				log.Printf("Запись лога: Введеные данные %s являются целым числом и переданы на 1 этап обработки.\n", command)
				source <- number
			}
		}
	}()
	log.Println("Запись лога: Запущена горутина - источник данных.")
	return source, done
}

// Первый этап пайплайна, отбраковывающий отрицательные числа
func NotNegativeNumbers(done <-chan bool, number <-chan int) <-chan int {
	notNegative := make(chan int)
	go func() {
		for {
			select {
			case checkNumber := <-number:
				if checkNumber >= 0 {
					log.Printf("Запись лога: Полученное неотрицательное число %d передано на следущий этап.", checkNumber)
					select {
					case notNegative <- checkNumber:
					case <-done:
						return
					}
				} else {
					log.Printf("Запись лога: Полученное отрицательное число %d отбраковано.", checkNumber)
				}
			case <-done:
				log.Println("Запись лога: 1 этап обработки получил сигнал о завершении работы.")
				return
			}
		}
	}()
	log.Println("Запись лога: Запущена горутина - 1 этап обработки данных: отбраковка отрицательных чисел.")
	return notNegative
}

// Второй этап пайплайна, фильтрующий числа не кратные 3-м и нули
func NotZeroAnd3Multi(done <-chan bool, number <-chan int) <-chan int {
	filtered := make(chan int)
	go func() {
		for {
			select {
			case checkNumber := <-number:
				if checkNumber%3 == 0 && checkNumber != 0 {
					log.Printf("Запись лога: Полученное число %d кратно 3-м и не равно нулю передано на следущий этап.", checkNumber)
					select {
					case filtered <- checkNumber:
					case <-done:
						log.Println("Запись лога: 2 этап обработки получил сигнал о завершении работы.")
						return
					}
				} else {
					if checkNumber%3 != 0 {
						log.Printf("Запись лога: Полученое некратное 3-м число %d отбраковано.", checkNumber)
					}
					if checkNumber == 0 {
						log.Println("Запись лога: Полученое число равное нулю отбраковано.")
					}
				}
			case <-done:
				log.Println("Запись лога: 2 этап обработки получил сигнал о завершении работы.")
				return
			}
		}
	}()
	log.Println("Запись лога: Запущена горутина - 2 этап обработки данных: отбраковка нулей и чисел некратных 3-м.")
	return filtered
}

// Третий этап пайплайна записывающий данные в фильтр и выдающий их по таймауту партией
// кроме каналов для работы принимает размер буфера и таймаут для отправки данных в секундах
func BufferAndTimeout(bSize int, bTimeout time.Duration, done <-chan bool, number <-chan int) <-chan []int {
	bufferChan := make(chan []int)
	buffer := ring.NewIntBuffer(bSize)
	go func() {
		for {
			select {
			case number2Buffer := <-number:
				err := buffer.Write(number2Buffer)
				if err != nil {
					log.Printf("При записи значения %d в буфер произошла ошибка: %v.\n", number2Buffer, err)
					fmt.Printf("При записи значения в буфер произошла ошибка: %v.\n", err)
				} else {
					log.Printf("Запись лога: В буфер записано значение %d.\n", number2Buffer)
				}
			case <-done:
				log.Println("Запись лога: 3 этап обработки: функция записи данных в буфер получил сигнал о завершении работы.")
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-time.After(bTimeout):
				numbersFromBuffer := buffer.Output()
				if len(numbersFromBuffer) > 0 {
					select {
					case bufferChan <- numbersFromBuffer:
						log.Printf("Запись лога: Из буфера прочитаны данные %v и отправлены конечному получателю данных.\n", numbersFromBuffer)
					case <-done:
						log.Println("Запись лога: 3 этап обработки: функция вывода данных из буфера получил сигнал о завершении работы.")
						return
					}
				} else {
					log.Printf("Запись лога: По истечению таймаута %v из буфера данные не прочитаны, он пуст", bTimeout)
				}
			case <-done:
				log.Println("Запись лога: 3 этап обработки: функция вывода данных из буфера получил сигнал о завершении работы.")
				return
			}
		}
	}()
	log.Println("Запись лога: Запущена горутина - 3 этап обработки данных: накопление данных в кольцевом буфере.")
	log.Printf("Запись лога: Создан кольцевой буфер размером %v с освобождениеем через %v.\n", bSize, bTimeout)
	return bufferChan
}

// Функция - конечный получатель отфильтрованных данных из буфера
func Receiver(done <-chan bool, buffer <-chan []int) {
	log.Println("Запись лога: Запущена горутина - последний этап обработки данных: вывод в консоль данных из буфера")
	for {
		select {
		case numbers := <-buffer:
			log.Printf("Запись лога: Получены данные из буфера: %v\n", numbers)
			fmt.Printf("Получены данные: %v\n", numbers)
		case <-done:
			log.Println("Запись лога: последний этап обработки: вывод в консоль данных из буфера получил сигнал о завершении работы.")
			return
		}
	}

}

const (
	bufferSize                  = 5
	bufferTimeout time.Duration = 10 * time.Second
)

func main() {
	log.Println("Запись лога: Pipeline запущен.")
	source, done := ConsoleInput()
	notNegativeChan := NotNegativeNumbers(done, source)
	notZeroAnd3Multi := NotZeroAnd3Multi(done, notNegativeChan)
	// Дополнительные параметры по сравнению с предыдущими этапами: размер буфера и время хранения данных
	fromBuffer := BufferAndTimeout(bufferSize, bufferTimeout, done, notZeroAnd3Multi)
	// Т.к. получатель не возвращает никаких каналов, то можно не оборачивать его код в анонимную горутину
	Receiver(done, fromBuffer)
	log.Println("Запись лога: Pipeline завершен.")
}
