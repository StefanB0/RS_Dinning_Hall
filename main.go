package main

import (
	"log"
	"restaurant/dinning-hall/pkg"
	"time"
)

const (
	KITCHEN_URL    = "http://kitchen:8881/order"
	LISTENPORT     = ":8882"
	NR_WAITERS     = 4
	NR_TABLES      = 10
	MAX_ORDER_DISH = 10
	MIN_W_DELAY    = 2
	MAX_W_DELAY    = 4
	RUNSPEED       = time.Millisecond * 20
)

var (
	tables   []pkg.Table
	waiters  []pkg.Waiter
	dishMenu []pkg.Dish
	globalOrderID = pkg.Counter{I: 0}	
	readyTables = make(chan int, NR_TABLES)
)

func initializeTables() {
	ledger = pkg.CreateRandomLedger(LEDGER_SIZE, MAX_ORDER_DISH, dishMenu)
	for i := 0; i < NR_TABLES; i++ {
		ntable := *pkg.NewTable(i, &globalOrderID, dishMenu, MAX_ORDER_DISH, RUNSPEED, readyTables)
		tables = append(tables, ntable)
		<-time.After(RUNSPEED * 23)
		go tables[i].WorkingTable()
	}
}

func initializeWaiters() {
	gRating := &pkg.GeneralRating{}

	for i := 0; i < NR_WAITERS; i++ {
		wchannel := make(chan pkg.OrderResponse, NR_TABLES)
		nwaiter := *pkg.NewWaiter(i, MIN_W_DELAY, MAX_W_DELAY, wchannel, readyTables, RUNSPEED, tables, dishMenu, gRating, KITCHEN_URL)
		waiters = append(waiters, nwaiter)
		go waiters[i].WorkingWaiter()
	}
}

func main() {
	log.Println("Hall take off!")
	dishMenu = pkg.ReadMenu("pkg/menu.json")
	initializeTables()
	initializeWaiters()
	pkg.StartServer(waiters, LISTENPORT)
}
