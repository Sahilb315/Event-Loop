package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Event struct {
	Key     string
	Data    string
	IsAsync bool
}

type EventLoop struct {
	Events          []Event
	Handlers        map[string]func(data string) string
	ProcessedEvents []EventResult
}

type EventResult struct {
	Key    string
	Result string
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		Handlers:        make(map[string]func(string) string),
		ProcessedEvents: []EventResult{},
		Events:          []Event{},
	}
}

// The on() method populates the handlers fields with an identifier
// for a given event and the code that should be executed in response to that event
func (e *EventLoop) on(key string, fn func(data string) string) *EventLoop {
	e.Handlers[key] = fn
	return e
}

// The dispatch() method is used scheduling/submitting the event for execution
func (e *EventLoop) dispatch(event Event) {
	e.Events = append(e.Events, event)
}

func (e *EventLoop) run() {
	if len(e.Events) != 0 {
		// Poll function implementation - Removing the the first elemt & returning it
		event := e.Events[0]
		e.Events = e.Events[1:]
		fmt.Printf("\nReceived Event: %s\n\n", event.Key)
		_, exists := e.Handlers[event.Key]
		if exists {
			startTime := time.Now()
			if event.IsAsync {
				e.processAsynchronously(event)
			} else {
				e.processSynchronously(event)
			}
			endTime := time.Since(startTime)
			fmt.Printf("Event loop was blocked for %v ms due to this operation\n\n", endTime.Milliseconds())
		} else {
			fmt.Printf("No handler found for %s\n\n", event.Key)
		}
	}
	if len(e.ProcessedEvents) != 0 {
		processedEvent := e.ProcessedEvents[0]
		e.ProcessedEvents = e.ProcessedEvents[1:]
		e.produceOutputFor(processedEvent)
	}
}

func (e *EventLoop) produceOutputFor(processedEvent EventResult) {
	fmt.Printf("Output for Event %q: %v\n\n", processedEvent.Key, processedEvent.Result)
}

func (e *EventLoop) processSynchronously(event Event) {
	handler := e.Handlers[event.Key]
	if handler != nil {
		result := handler(event.Data)
		e.produceOutputFor(EventResult{
			Key:    event.Key,
			Result: result,
		})
	}
}

func (e *EventLoop) processAsynchronously(event Event) {
	go func() {
		handler := e.Handlers[event.Key]
		if handler != nil {
			result := handler(event.Data)
			e.ProcessedEvents = append(e.ProcessedEvents, EventResult{
				Key:    event.Key,
				Result: result,
			})
		}
	}()
}

func readFile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		exists := os.IsExist(err)
		fmt.Printf("Exists: %v\n", exists)
		if !exists {
			file, err := os.Create(filename)
			file.WriteString("New file created")
			if err != nil {
				return fmt.Sprintf("Error creating file: %s", err)
			}
			file.Seek(0, 0)
			data, err := io.ReadAll(file)
			if err != nil {
				return fmt.Sprintf("Error reading file: %s", err)
			}
			return string(data)
		}
		return fmt.Sprintf("Error reading file: %s", err)
	}
	return string(data)
}

func fetchDataFromAPI(id string) string {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/posts/" + id)
	if err != nil {
		return "Error fetching data from API"
	}
	var data struct {
		ID     int    `json:"id"`
		UserID int    `json:"userId"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "Error decoding data from API"
	}
	return fmt.Sprintf("Fetched post from API: %v", data)
}

func isAsync(choice string) bool {
	return choice == "2"
}

func generateUniqueEventKey(base string, id int) string {
	return fmt.Sprintf("%s-%d", base, id)
}

func main() {
	eventLoop := NewEventLoop()
	reader := bufio.NewReader(os.Stdin)
	utils := struct {
		generateUniqueEventKey func(string, int) string
		readFile               func(string) string
		fetchDataFromAPI       func(string) string
		isAsync                func(string) bool
	}{
		generateUniqueEventKey: generateUniqueEventKey,
		readFile:               readFile,
		fetchDataFromAPI:       fetchDataFromAPI,
		isAsync:                isAsync,
	}

	eventID := 0

	for {
		var usersChoice string
		for {
			fmt.Println("What kind of task would you like to submit to the Event Loop?")
			fmt.Println(" 1. Wish me Hello")
			fmt.Println(" 2. Print the contents of a file named hello.txt")
			fmt.Println(" 3. Retrieve data from API & print it")
			fmt.Println(" 4. Print output of previously submitted Async task")
			fmt.Println(" 5. Exit!")
			fmt.Print(" > ")

			usersChoice, _ = reader.ReadString('\n')
			usersChoice = strings.TrimSpace(usersChoice)
			if usersChoice != "" && (usersChoice >= "1" && usersChoice <= "5") {
				break
			}
			fmt.Println("Invalid input. Please select a valid option (1-5).")
		}

		if usersChoice == "5" {
			break
		}

		var operationType string
		if usersChoice != "4" {
			for {
				fmt.Println("How would you like to execute this operation?")
				fmt.Println(" 1. Synchronously (this would block the Event Loop until the operation completes)")
				fmt.Println(" 2. Asynchronously (this won't block Event Loop in any way)")
				fmt.Print(" > ")

				operationType, _ = reader.ReadString('\n')
				operationType = strings.TrimSpace(operationType)

				if operationType != "" && (operationType == "1" || operationType == "2") {
					break
				}
				fmt.Println("Invalid input. Please select a valid option (1 or 2).")
			}

		}
		isAsync := utils.isAsync(operationType)
		switch usersChoice {
		case "1":
			uniqueEventKey := utils.generateUniqueEventKey("hello", eventID)
			eventID++
			eventLoop.on(uniqueEventKey, func(data string) string {
				return fmt.Sprintf("Hello! %s", data)
			}).dispatch(Event{
				Key:     uniqueEventKey,
				Data:    "How are you doing today?",
				IsAsync: isAsync,
			})
		case "2":
			uniqueEventKey := utils.generateUniqueEventKey("read-file", eventID)
			eventID++
			eventLoop.on(uniqueEventKey, utils.readFile).dispatch(Event{
				Key:     uniqueEventKey,
				Data:    "hello.txt",
				IsAsync: isAsync,
			})

		case "3":
			uniqueEventKey := utils.generateUniqueEventKey("fetch-from-api", eventID)
			eventID++
			eventLoop.on(uniqueEventKey, utils.fetchDataFromAPI).dispatch(Event{
				Key:     uniqueEventKey,
				Data:    "2",
				IsAsync: isAsync,
			})
		case "4":
		}
		eventLoop.run()
	}
}
