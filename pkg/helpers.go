package pkg

import (
	"math/rand"
	"sync"
	"time"
)

type Counter struct {
	I  int
	mu sync.Mutex
}

func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.I++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.I
}

func DetermineRating(maxWait int, pickUpTime, servingTime time.Time, RUNSPEED time.Duration, correctDelivery bool) int {
	rating:= 0

	if !correctDelivery {
		return 0
	}

	switch {
	case int(servingTime.Sub(pickUpTime) / RUNSPEED) < maxWait:
		rating = 5
	case int(servingTime.Sub(pickUpTime) / RUNSPEED) < (maxWait * 11) / 10:
		rating = 4
	case int(servingTime.Sub(pickUpTime) / RUNSPEED) < (maxWait * 12) / 10:
		rating = 3
	case int(servingTime.Sub(pickUpTime) / RUNSPEED) < (maxWait * 13) / 10:
		rating = 2
	case int(servingTime.Sub(pickUpTime) / RUNSPEED) < (maxWait * 14) / 10:
		rating = 1
	default:
		rating = 0
	}

	return rating
}

func CheckMatchingOrders(order Order, orderResponse OrderResponse) bool {
	if order.OrderID != orderResponse.OrderID ||
		order.TableID != orderResponse.TableID ||
		order.WaiterID != orderResponse.WaiterID ||
		order.Priority != orderResponse.Priority ||
		order.MaxWait != orderResponse.MaxWait ||
		order.PickUpTime != orderResponse.PickUpTime ||
		!slicesEqual(order.Items, orderResponse.Items) {
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

func Delay(min, max int, RUNSPEED time.Duration) {
	stime := time.Duration(rand.Intn(max-min+1)+min) * RUNSPEED
	time.Sleep(stime)
}