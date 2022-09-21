package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"restaurant/dinning-hall/pkg"

	"github.com/gorilla/mux"
)

const (
	KITCHEN_URL = "http://kitchen:8881/order"
	LISTENPORT  = ":8882"
	NR_WAITERS  = 4
	NR_TABLES   = 10
	RUNSPEED    = time.Millisecond
	MIN_W_DELAY = 2
	MAX_W_DELAY = 4
)

const (
	FREE = iota
	W_ORDER
	W_SERVED
)

var (
	tables  [NR_TABLES]Table
	waiters [NR_WAITERS]Waiter

	globalOrderID = pkg.Counter{I: 0}

	readyTables = make(chan int, NR_TABLES)

	waiterLog        *log.Logger
	orderLog         *log.Logger
	communicationLog *log.Logger
	errorLog         *log.Logger
)

type MyServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}

type Table struct {
	tableID    int
	tableState int
	order      pkg.Order
}

type Waiter struct {
	waiterID      int
	waiterChannel chan pkg.OrderResponse
}

type ServedOrder struct {
	orderResponse   pkg.OrderResponse
	servingTime     time.Time
	correctDelivery bool
	rating          int
}

func NewServer() *MyServer {
	//create server
	s := &MyServer{
		Server: http.Server{
			Addr:         LISTENPORT,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		shutdownReq: make(chan bool),
	}

	router := mux.NewRouter()

	//register handlers
	router.HandleFunc("/shutdown", s.ShutdownHandler)
	router.HandleFunc("/distribution", receiveRequest).Methods("POST")

	//set http server handler
	s.Handler = router

	return s
}

func (s *MyServer) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	//Wait interrupt or shutdown request through /shutdown
	select {
	case sig := <-irqSig:
		log.Printf("Shutdown request (signal: %v)", sig)
	case sig := <-s.shutdownReq:
		log.Printf("Shutdown request (/shutdown %v)", sig)
	}

	log.Printf("Stoping http server ...")

	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//shutdown the server
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
}

func (s *MyServer) ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Shutdown server"))

	//Do nothing if shutdown request already issued
	//if s.reqCount == 0 then set to 1, return true otherwise false
	if !atomic.CompareAndSwapUint32(&s.reqCount, 0, 1) {
		log.Printf("Shutdown through API call in progress...")
		return
	}

	go func() {
		s.shutdownReq <- true
	}()
}

func receiveRequest(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Invalid request")
	}

	var newFinishedDish pkg.OrderResponse
	json.Unmarshal(reqBody, &newFinishedDish)

	log.Println("Order received, id:", newFinishedDish.OrderID)
	waiters[newFinishedDish.WaiterID].waiterChannel <- newFinishedDish
	w.WriteHeader(http.StatusCreated)
}

func startServer() {
	server := NewServer()

	done := make(chan bool)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Listen and serve: %v", err)
		}
		done <- true
	}()

	//wait shutdown
	server.WaitShutdown()

	<-done
	log.Printf("DONE!")
}

func initializeTables() {
	for i := 0; i < NR_TABLES; i++ {
		tables[i].tableID = i
		tables[i].tableState = FREE
	}
}

func initializeWaiters() {
	for i := 0; i < NR_WAITERS; i++ {
		waiters[i].waiterChannel = make(chan pkg.OrderResponse, NR_TABLES)
		waiters[i].waiterID = i
		go waiters[i].workingWaiter()
	}
}

func initLogs() {
	waiterFile, err1 := os.OpenFile("logs/waiter_logs.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	orderFile, err2 := os.OpenFile("logs/order_logs.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	communicationFile, err3 := os.OpenFile("logs/communication_logs.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	errorFile, err4 := os.OpenFile("logs/error_logs.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		log.Fatal(err1, err2, err3, err4)
	}

	waiterLog = log.New(waiterFile, "Waiter: ", log.Ltime|log.Lmicroseconds|log.Lshortfile)
	orderLog = log.New(orderFile, "Order: ", log.Ltime|log.Lmicroseconds|log.Lshortfile)
	communicationLog = log.New(communicationFile, "Communication: ", log.Ltime|log.Lmicroseconds|log.Lshortfile)
	errorLog = log.New(errorFile, "error: ", log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func (w *Waiter) workingWaiter() {
	for {
		select {
		case t := <-readyTables:
			w.takeOrder(t)

		case returnOrder := <-w.waiterChannel:
			w.returnOrderToTable(returnOrder)
		}
	}
}

func (w *Waiter) takeOrder(tableID int) {
	tables[tableID].order = *tables[tableID].createNewOrder()
	tables[tableID].order.PickUpTime = time.Now().Round(RUNSPEED)
	tables[tableID].order.WaiterID = w.waiterID
	tables[tableID].tableState = W_SERVED

	pkg.Delay(MIN_W_DELAY, MAX_W_DELAY, RUNSPEED)

	sendOrderKitchen(tables[tableID].order)
}

func (w *Waiter) returnOrderToTable(returnOrder pkg.OrderResponse) {
	servingTime := time.Now().Round(RUNSPEED)
	correctDelivery := pkg.CheckMatchingOrders(tables[returnOrder.TableID].order, returnOrder)
	rating := pkg.DetermineRating(
		returnOrder.MaxWait,
		returnOrder.PickUpTime,
		servingTime,
		RUNSPEED,
		correctDelivery,
	)

	tables[returnOrder.TableID].order = pkg.Order{}
	tables[returnOrder.TableID].tableState = FREE

	log.Println("Order:", returnOrder.OrderID, "Match order:", correctDelivery, "Waiter:", w.waiterID, "Rating:", rating)
}

func (t *Table) createNewOrder() *pkg.Order {
	globalOrderID.Increment()
	max := 0
	_items := []int{}
	itemsNr := rand.Intn(10) + 1

	for i := 0; i < itemsNr; i++ {
		_items = append(_items, rand.Intn(12)+1)
		if max < pkg.DishMenu[_items[i]].PreparationTime {
			max = pkg.DishMenu[_items[i]].PreparationTime
		}
	}

	newOrder := &pkg.Order{
		OrderID:  globalOrderID.Value(),
		TableID:  t.tableID,
		Items:    _items,
		Priority: 4,
		MaxWait:  max * 13 / 10,
	}

	return newOrder
}

func sendOrderKitchen(order pkg.Order) {
	log.Println("Waiter:", order.WaiterID, "order:", order.OrderID, "sent to kitchen")

	payloadBuffer := new(bytes.Buffer)
	json.NewEncoder(payloadBuffer).Encode(order)

	req, _ := http.NewRequest("POST", KITCHEN_URL, payloadBuffer)
	client := &http.Client{}
	client.Do(req)
}

func Lounge() {
	for {
		for i := range tables {
			x := i
			if tables[i].tableState == FREE {
				tables[i].tableState = W_ORDER
				go func() {
					pkg.Delay(10, 30, RUNSPEED)
					readyTables <- x
				}()
			}
		}
	}
}

func main() {
	time.Sleep(time.Second)
	initializeTables()
	initializeWaiters()
	// initLogs()
	go Lounge()
	startServer()
}
