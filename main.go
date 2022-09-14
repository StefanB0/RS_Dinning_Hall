package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"

	// "os"
	"time"

	"restaurant/dinning-hall/resources"

	"github.com/gorilla/mux"
)

const nrWaiters = 4
const nrTables = 10
const runSpeed = time.Millisecond
const kitchenUrl = "https://b83a7d0f-80c5-423c-82bb-4886ffe3a7e6.mock.pstmn.io/order"

var tables [nrTables]table
var logs []servedOrder
var logsChannel = make(chan servedOrder, nrTables)
var waiterNotice [nrWaiters]chan Response

var readyTables = make(chan int, nrTables)

const (
	FREE = iota
	W_ORDER
	W_SERVED
)

type table struct {
	tableID    int
	tableState int
	tableTimer time.Time
	_order     Order
}

type waiter struct {
	waiterID    int
	waitingList []int
}

type Order struct {
	OrderID    int       `json:"order_id"`
	TableID    int       `json:"table_id"`
	WaiterID   int       `json:"waiter_id"`
	Items      []int     `json:"items"`
	Priority   int       `json:"priority"`
	MaxWait    int       `json:"max_wait"`
	PickUpTime time.Time `json:"pick_up_time"`
}

type Response struct {
	OrderID        int       `json:"order_id"`
	TableID        int       `json:"table_id"`
	WaiterID       int       `json:"waiter_id"`
	Items          []int     `json:"items"`
	Priority       int       `json:"priority"`
	MaxWait        int       `json:"max_wait"`
	PickUpTime     time.Time `json:"pick_up_time"`
	CookingTime    int       `json:"cooking_time"`
	CookingDetails []struct {
		Cook_ID int
		Food_ID int
	} `json:"cooking_details"`
}

type servedOrder struct {
	_response       Response
	servingTime     time.Time
	correctDelivery bool
	rating          int
}

func initializeTables() {
	for i := 0; i < nrTables; i++ {
		tables[i].tableID = i
		// tables[i].tableTimer = time.Now().Add(time.Duration((rand.Intn(20))+10) * runSpeed)
		tables[i].tableTimer = time.Now()
		tables[i].tableState = FREE
	}
}

func initializeWaiters() {
	for i := 0; i < nrWaiters; i++ {
		waiterNotice[i] = make(chan Response, nrTables)
		go workingWaiter(i, waiterNotice)
	}
}

func workingWaiter(id int, waiterNotice [nrWaiters]chan Response) {
	parameters := &waiter{
		waiterID:    id,
		waitingList: []int{},
	}

	for {
		select {
		case t := <-readyTables:
			time.Sleep(time.Duration(rand.Intn(2)+2) * runSpeed)
			parameters.waitingList = append(parameters.waitingList, t)
			tables[t]._order.WaiterID = parameters.waiterID
			tables[t]._order.PickUpTime = time.Now()
			sendOrderKitchen(tables[t]._order)
			fmt.Println("waiter ", parameters.waiterID, " picked up the order from table ", t, " and sent it to the kitchen")

		case returnOrder := <-waiterNotice[parameters.waiterID]:

			log := &servedOrder{
				_response:   returnOrder,
				servingTime: time.Now(),
			}
			if !checkMatchingOrders(tables[returnOrder.TableID]._order, returnOrder) {
				fmt.Println("WRONG ORDER at table ", returnOrder.TableID, ", order ID ", returnOrder.OrderID, ", waiter ", parameters.waiterID, "!!!")
				log.correctDelivery = false
				log.rating = 0
			} else {
				log.correctDelivery = true
			}

			tables[returnOrder.TableID]._order = Order{}
			tables[returnOrder.TableID].tableTimer = time.Now().Add(time.Duration((rand.Intn(20))+10) * runSpeed)
			tables[returnOrder.TableID].tableState = FREE

			fmt.Println(log)
			logsChannel <- *log
		}

	}
}

func sendOrderKitchen(_order Order) {
	payloadBuffer := new(bytes.Buffer)
	json.NewEncoder(payloadBuffer).Encode(_order)
	req, _ := http.NewRequest("POST", kitchenUrl, payloadBuffer)
	client := &http.Client{}
	client.Do(req)
	println("order sent to kitchen?")
}

func checkMatchingOrders(_order Order, _response Response) bool {
	if _order.OrderID != _response.OrderID ||
		_order.TableID != _response.TableID ||
		_order.WaiterID != _response.WaiterID ||
		_order.Priority != _response.Priority ||
		_order.MaxWait != _response.MaxWait ||
		_order.PickUpTime != _response.PickUpTime ||
		slicesEqual(_order.Items, _response.Items) {
		return false
	}
	return true
}

func slicesEqual(sa []int, sb []int) bool {
	if len(sa) != len(sb) {
		return false
	}

	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}

	return true
}

func remove(s []int, i int) []int {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func Lounge() {
	initializeTables()
	_orderID := 0
	for {
		for i := range tables {
			if tables[i].tableState == FREE /*&& tables[i].tableTimer.After(time.Now())*/ {
				tables[i].tableState = W_ORDER
				tables[i]._order = *createNewOrder(&_orderID, i)

				readyTables <- i

			}
		}
	}
}

func createNewOrder(orderID *int, tableID int) *Order {
	*orderID++
	max := 0
	_items := []int{}
	itemsNr := rand.Intn(10) + 1

	for i := 0; i < itemsNr; i++ {
		_items = append(_items, rand.Intn(13))
		if max < resources.DishMenu[_items[i]].PreparationTime {
			max = resources.DishMenu[_items[i]].PreparationTime
		}
	}

	newOrder := &Order{
		OrderID:  *orderID,
		TableID:  tableID,
		Items:    _items,
		Priority: 4,
		MaxWait:  max * 13 / 10,
	}

	return newOrder
}

func logsWriter() {
	for {
		l := <-logsChannel
		logs = append(logs, l)
	}
}

func receiveRequest(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Invalid request")
	}

	var newFinishedDish Response
	json.Unmarshal(reqBody, &newFinishedDish)

	fmt.Println(newFinishedDish)
	waiterNotice[newFinishedDish.WaiterID] <- newFinishedDish
	w.WriteHeader(http.StatusCreated)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("HomePageAccessed")
}

func main() {
	initializeWaiters()
	go Lounge()

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homePage)
	router.HandleFunc("/distribution", receiveRequest).Methods("POST")

	log.Fatal(http.ListenAndServe(":8086", router))
}
