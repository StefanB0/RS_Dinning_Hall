package pkg

import (
	"math/rand"
	"time"
)

type Table struct {
	tableID       int
	tableState    int
	globalOrderID *Counter
	dishMenu      []Dish
	ledger        []float64
	order         Order
	maxOrderNr    int
	runspeed      time.Duration
	readyTables   chan int
}

const (
	FREE = iota
	W_ORDER
	W_SERVED
)

func NewTable(
	_tableID int,
	_globalOrderID *Counter,
	_dishMenu []Dish,
	_ledger []float64,
	_maxOrderNr int,
	_runspeed time.Duration,
	_readyTables chan int,
) *Table {
	return &Table{
		tableID:       _tableID,
		tableState:    FREE,
		globalOrderID: _globalOrderID,
		dishMenu:      _dishMenu,
		ledger:        _ledger,
		maxOrderNr:    _maxOrderNr,
		runspeed:      _runspeed,
		readyTables:   _readyTables,
	}
}

func (t *Table) createNewOrder() *Order {
	t.globalOrderID.Increment()
	max := 0
	_items := []int{}
	itemsNr := rand.Intn(t.maxOrderNr) + 1

	for i := 0; i < itemsNr; i++ {
		_items = append(_items, rand.Intn(len(t.dishMenu)-1)+1)
		if max < t.dishMenu[_items[i]].PreparationTime {
			max = t.dishMenu[_items[i]].PreparationTime
		}
	}

	newOrder := &Order{
		OrderID: t.globalOrderID.Value(),
		TableID: t.tableID,
		Items:   _items,
		MaxWait: max * 13 / 10,
	}

	newOrder.Priority = CalculatePriority(newOrder, t.dishMenu, t.ledger)

	return newOrder
}

func (t *Table) WorkingTable() {
	for {
		if t.tableState == FREE {
			<-time.After(30 * t.runspeed)
			t.tableState = W_ORDER
			t.readyTables <- t.tableID
		}
	}
}
