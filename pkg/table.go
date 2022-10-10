package pkg

import (
	"time"
)

type Table struct {
	tableID       int
	tableState    int
	globalOrderID *Counter
	dishMenu      []Dish
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
	_maxOrderNr int,
	_runspeed time.Duration,
	_readyTables chan int,
) *Table {
	return &Table{
		tableID:       _tableID,
		tableState:    FREE,
		globalOrderID: _globalOrderID,
		dishMenu:      _dishMenu,
		maxOrderNr:    _maxOrderNr,
		runspeed:      _runspeed,
		readyTables:   _readyTables,
	}
}

func (t *Table) createNewOrder() *Order {
	id := t.globalOrderID.Increment()
	max := 0
	_items := []int{}
	itemsNr := lengthDistribution(t.maxOrderNr)

	for i := 0; i < itemsNr; i++ {
		_items = append(_items, menuDistribution())
		if max < t.dishMenu[_items[i]].PreparationTime {
			max = t.dishMenu[_items[i]].PreparationTime
		}
	}

	newOrder := &Order{
		OrderID: id,
		TableID: t.tableID,
		Items:   _items,
		MaxWait: max * 13 / 10,
	}

	newOrder.Priority = SimplePriority(newOrder, t.maxOrderNr)

	return newOrder
}

func (t *Table) WorkingTable() {
	Delay(10, 100, t.runspeed)
	for {
		if t.tableState == FREE {
			Delay(45, 60, t.runspeed)
			t.tableState = W_ORDER
			t.readyTables <- t.tableID
		}
	}
}
