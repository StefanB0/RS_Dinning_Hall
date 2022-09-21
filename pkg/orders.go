package pkg

import "time"

type Order struct {
	OrderID    int       `json:"order_id"`
	TableID    int       `json:"table_id"`
	WaiterID   int       `json:"waiter_id"`
	Items      []int     `json:"items"`
	Priority   int       `json:"priority"`
	MaxWait    int       `json:"max_wait"`
	PickUpTime time.Time `json:"pick_up_time"`
}

type OrderResponse struct {
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