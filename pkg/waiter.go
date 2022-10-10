package pkg

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Waiter struct {
	waiterID      int
	minDelay      int
	maxDelay      int
	waiterChannel chan OrderResponse
	readyTables   chan int
	runspeed      time.Duration
	tables        []Table
	dishmenu      []Dish
	gRating       *GeneralRating
	kitchenURL    string
}

func NewWaiter(
	_waiterID int,
	_minDelay int,
	_maxDelay int,
	_waiterChannel chan OrderResponse,
	_readyTables chan int,
	_runspeed time.Duration,
	_tables []Table,
	_dishmenu []Dish,
	_gRating *GeneralRating,
	_kitchenURL string,
) *Waiter {
	return &Waiter{
		waiterID:      _waiterID,
		minDelay:      _minDelay,
		maxDelay:      _maxDelay,
		waiterChannel: _waiterChannel,
		readyTables:   _readyTables,
		runspeed:      _runspeed,
		tables:        _tables,
		dishmenu:      _dishmenu,
		gRating:       _gRating,
		kitchenURL:    _kitchenURL,
	}
}

func (w *Waiter) WorkingWaiter() {
	for {
		select {
		case t := <-w.readyTables:
			w.takeOrder(t)
		case returnOrder := <-w.waiterChannel:
			w.returnOrderToTable(returnOrder)
		}
	}
}

func (w *Waiter) takeOrder(tableID int) {
	w.tables[tableID].order = *w.tables[tableID].createNewOrder()
	w.tables[tableID].order.PickUpTime = time.Now().Round(w.runspeed)
	w.tables[tableID].order.WaiterID = w.waiterID
	w.tables[tableID].tableState = W_SERVED

	time.Sleep(2 * w.runspeed)

	sendOrderKitchen(w.tables[tableID].order, w.kitchenURL)
}

func sendOrderKitchen(order Order, url string) {
	payloadBuffer := new(bytes.Buffer)
	json.NewEncoder(payloadBuffer).Encode(order)

	log.Println("HOS:", order.OrderID)
	req, _ := http.NewRequest("POST", url, payloadBuffer)
	client := &http.Client{}
	client.Do(req)
}

func (w *Waiter) returnOrderToTable(returnOrder OrderResponse) {
	returnOrder.PickUpTime = returnOrder.PickUpTime.Round(w.runspeed)
	servingTime := time.Now().Round(w.runspeed)
	correctDelivery := CheckMatchingOrders(w.tables[returnOrder.TableID].order, returnOrder)
	durations := listDurations(returnOrder.Items, w.dishmenu)
	rating := DetermineRating(
		returnOrder.MaxWait,
		returnOrder.PickUpTime,
		servingTime,
		w.runspeed,
		correctDelivery,
	)

	avgR := w.gRating.Increment(int(rating))
	log.Println("HOD: OrderID:", returnOrder.OrderID, durations, "P:", returnOrder.Priority, "T:", returnOrder.CookingTime, ":", int(servingTime.Sub(returnOrder.PickUpTime)/w.runspeed), "/", returnOrder.MaxWait, "R:", rating, "Avg:", avgR)

	if !correctDelivery {
		log.Println(
			"ERROR: ",
			"Order ID:", returnOrder.OrderID, "Table:", returnOrder.TableID, "Waiter:", w.waiterID, "\n",
			"Returned/Expected\n",
			"Order ID:", returnOrder.OrderID, ":", w.tables[returnOrder.TableID].order.OrderID, returnOrder.OrderID == w.tables[returnOrder.TableID].order.OrderID, "\n",
			"Waiter ID:", returnOrder.WaiterID, ":", w.tables[returnOrder.TableID].order.WaiterID, returnOrder.WaiterID == w.tables[returnOrder.TableID].order.WaiterID, "\n",
			"Table ID:", returnOrder.TableID, ":", w.tables[returnOrder.TableID].order.TableID, returnOrder.TableID == w.tables[returnOrder.TableID].order.TableID, "\n",
			"Items ID:", returnOrder.Items, ":", w.tables[returnOrder.TableID].order.Items, SlicesEqual(returnOrder.Items, w.tables[returnOrder.TableID].order.Items), "\n",
			"Priority ID:", returnOrder.Priority, ":", w.tables[returnOrder.TableID].order.Priority, returnOrder.Priority == w.tables[returnOrder.TableID].order.Priority, "\n",
			"Max Wait ID:", returnOrder.MaxWait, ":", w.tables[returnOrder.TableID].order.MaxWait, returnOrder.MaxWait == w.tables[returnOrder.TableID].order.MaxWait, "\n",
			"Pick-up time ID:", returnOrder.PickUpTime, ":", w.tables[returnOrder.TableID].order.PickUpTime, returnOrder.PickUpTime.Local().Equal(w.tables[returnOrder.TableID].order.PickUpTime),
		)
	}
	w.tables[returnOrder.TableID].order = Order{}
	w.tables[returnOrder.TableID].tableState = FREE
}
