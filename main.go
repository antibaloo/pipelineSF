package main

import (
	"bufio"
	"fmt"
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
			command, _ := reader.ReadString('\n')
			command = strings.Replace(command, "\r", "", -1)
			command = strings.Replace(command, "\n", "", -1)
			switch command {
			case "exit":
				fmt.Println("Программа завершила работу!")
				return
			default:
				number, err := strconv.Atoi(command)
				if err != nil {
					fmt.Println("Программа принимает к обработке только целые числа!")
					continue
				}
				source <- number
			}
		}
	}()
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
					select {
					case notNegative <- checkNumber:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return notNegative
}

// Второй этап пайплайна, фильтрующий числа не кратрные 3-м и нули
func NotZeroAnd3Multi(done <-chan bool, number <-chan int) <-chan int {
	filtered := make(chan int)
	go func() {
		for {
			select {
			case checkNumber := <-number:
				if checkNumber%3 == 0 && checkNumber != 0 {
					select {
					case filtered <- checkNumber:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return filtered
}

// Третий этап пайплайна записывающий данные в фильтр и выдающий их по таймайту партией
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
					fmt.Printf("При записи значения в буфер произошла ошибка: %v\n", err)
				}
			case <-done:
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
					case <-done:
						return
					}

				}
			case <-done:
				return
			}
		}
	}()

	return bufferChan
}

// Функция - конечный получатель отфильтрованных данных из буфера
func Receiver(done <-chan bool, buffer <-chan []int) {
	for {
		select {
		case numbers := <-buffer:
			fmt.Printf("Получены данные: %v\n", numbers)
		case <-done:
			return
		}
	}
}

const (
	bufferSize                  = 5
	bufferTimeout time.Duration = 10 * time.Second
)

func main() {

	source, done := ConsoleInput()
	notNegativeChan := NotNegativeNumbers(done, source)
	notZeroAnd3Multi := NotZeroAnd3Multi(done, notNegativeChan)
	// Дополнительные параметры по сравнению с предыдущими этапами: размер буфера и время хранения данных
	fromBuffer := BufferAndTimeout(bufferSize, bufferTimeout, done, notZeroAnd3Multi)
	// Т.к. получатель не возвращает никаких каналов, то можно не оборачивать его код в анонимную горутину
	Receiver(done, fromBuffer)
}
