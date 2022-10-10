package pkg

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"sort"
	"time"
)

type Counter struct {
	I  int
	mu sync.Mutex
}

type GeneralRating struct {
	totalRating float32
	totalOrders float32
	avgRating   float32
	mu          sync.Mutex
}

func (r *GeneralRating) Increment(rating int) float32 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.totalOrders++
	r.totalRating += float32(rating)
	r.avgRating = r.totalRating / r.totalOrders

	return r.avgRating
}

func (c *Counter) Increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.I++

	return c.I
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.I
}

func DetermineRating(maxWait int, pickUpTime, servingTime time.Time, RUNSPEED time.Duration, correctDelivery bool) float64 {
	rating := 0.0
	elapsedTime := int(servingTime.Sub(pickUpTime) / RUNSPEED)

	if !correctDelivery {
		return 0
	}

	switch {
	case elapsedTime < maxWait:
		rating = 5
	case elapsedTime < (maxWait*11)/10:
		rating = 4
	case elapsedTime < (maxWait*12)/10:
		rating = 3
	case elapsedTime < (maxWait*13)/10:
		rating = 2
	case elapsedTime < (maxWait*14)/10:
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

		!order.PickUpTime.Local().Equal(orderResponse.PickUpTime) ||
		!SlicesEqual(order.Items, orderResponse.Items) {
		return false
	}

	return true
}

func SimplePriority(newOrder *Order, maxOrderNr int) int {
	r := (maxOrderNr-1)/5 + 1
	return (len(newOrder.Items)-1)/r + 1
}

func SlicesEqual(sa []int, sb []int) bool {
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

func ReadMenu(path string) []Dish {
	jsonfile, err := os.Open(path)
	defer jsonfile.Close()

	if err != nil {
		log.Println(err)
	}

	bytevalue, _ := ioutil.ReadAll(jsonfile)
	newMenu := []Dish{}
	json.Unmarshal(bytevalue, &newMenu)

	return newMenu
}

func listDurations(slice []int, dishmenu []Dish) []int {
	s := make([]int, len(slice))
	for i := 0; i < len(slice); i++ {
		s[i] = dishmenu[slice[i]].PreparationTime
	}
	return s
}

func menuDistribution() int {
	result := 0
	weights := []int{2, 4, 1, 4, 2, 1, 2, 2, 1, 4, 4, 4, 2}
	for i := 1; i < len(weights); i++ {
		weights[i] += weights[i-1]
	}
	tR := weights[len(weights)-1]
	r := rand.Intn(tR) + 1
	for i := 1; i < len(weights); i++ {
		if r > weights[i-1] && r <= weights[i] {
			result = i
		}
	}
	return result
}

func lengthDistribution(max int) int {
	result := 0
	r := rand.Intn(4)
	switch r {
	case 0:
		result = rand.Intn(3) + 1
	case 1:
		result = rand.Intn(5) + 1
	case 2:
		result = rand.Intn(7) + 1
	case 3:
		result = rand.Intn(max) + 1
	}

	if result > max {
		result = max
	}

	return result
}
