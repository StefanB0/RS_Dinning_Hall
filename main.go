package main

import (
	"restaurant/dinning-hall/pkg"
	"time"
)

const (
	KITCHEN_URL    = "http://kitchen:8881/order"
	LISTENPORT     = ":8882"
	NR_WAITERS     = 4
	NR_TABLES      = 10
	MAX_ORDER_DISH = 5
	LEDGER_SIZE    = 100
	MIN_W_DELAY    = 2
	MAX_W_DELAY    = 4
	RUNSPEED       = time.Millisecond
)

var (
	tables   []pkg.Table
	waiters  []pkg.Waiter
	dishMenu []pkg.Dish
	ledger   []float64

	globalOrderID = pkg.Counter{I: 0}

	readyTables = make(chan int, NR_TABLES)
)

func initializeTables() {
	ledger = pkg.CreateRandomLedger(LEDGER_SIZE, MAX_ORDER_DISH, dishMenu)
	for i := 0; i < NR_TABLES; i++ {
		ntable := *pkg.NewTable(i, &globalOrderID, dishMenu, ledger, MAX_ORDER_DISH, RUNSPEED, readyTables)
		tables = append(tables, ntable)
		<-time.After(RUNSPEED * 23)
		go tables[i].WorkingTable()
	}
}

func initializeWaiters() {
	for i := 0; i < NR_WAITERS; i++ {
		wchannel := make(chan pkg.OrderResponse, NR_TABLES)
		nwaiter := *pkg.NewWaiter(i, MIN_W_DELAY, MAX_W_DELAY, wchannel, readyTables, RUNSPEED, tables, KITCHEN_URL)
		waiters = append(waiters, nwaiter)
		go waiters[i].WorkingWaiter()
	}
}

func main() {
	<-time.After(time.Second)
	dishMenu = pkg.ReadMenu("pkg/menu.json")
	initializeTables()
	initializeWaiters()
	pkg.StartServer(waiters, LISTENPORT)
}
