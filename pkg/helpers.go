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

func DetermineRating(maxWait int, pickUpTime, servingTime time.Time, RUNSPEED time.Duration, correctDelivery bool) float64 {
	rating := 0.0

	if !correctDelivery {
		return 0
	}

	switch {
	case int(servingTime.Sub(pickUpTime)/RUNSPEED) < maxWait:
		rating = 5
	case int(servingTime.Sub(pickUpTime)/RUNSPEED) < (maxWait*11)/10:
		rating = 4
	case int(servingTime.Sub(pickUpTime)/RUNSPEED) < (maxWait*12)/10:
		rating = 3
	case int(servingTime.Sub(pickUpTime)/RUNSPEED) < (maxWait*13)/10:
		rating = 2
	case int(servingTime.Sub(pickUpTime)/RUNSPEED) < (maxWait*14)/10:
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

func CalculatePriority(newOrder *Order, dishMenu []Dish, ledger []float64) int {
	var totalTime float64
	newOrderPosition := len(ledger) - 1
	for _, id := range newOrder.Items {
		totalTime += float64(dishMenu[id].PreparationTime)
	}
	
	fractionalPriority := float64(newOrder.MaxWait) / totalTime
	for i := 1; i < len(ledger); i++ {
		if (fractionalPriority < ledger[i]) {
			newOrderPosition = i - 1
			break
		}
	}

	ledger[newOrderPosition] = fractionalPriority
	ledgerRange := len(ledger) / 5

	return (newOrderPosition / ledgerRange) + 1
}

func CreateRandomLedger(size int, maxDish int, dishMenu []Dish) []float64 {
	ledger := make([]float64, size)
	for i := 0; i < size; i++ {
		itemsNr := rand.Intn(maxDish) + 1
		totalTime := 0
		maxTime := 0
		for i := 0; i < itemsNr; i++ {
			dishTime := dishMenu[rand.Intn(len(dishMenu)-1)+1].PreparationTime
			totalTime += dishTime
			if (maxTime < dishTime) {
				maxTime = dishTime
			}
		}
		ledger[i] = float64(maxTime) / float64(totalTime)
	}

	sort.Float64s(ledger)
	return ledger
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
